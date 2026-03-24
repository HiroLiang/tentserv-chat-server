package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/cache"
)

const historyTTL = 24 * time.Hour

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrUpstreamUnavailable = errors.New("upstream unavailable")
)

type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []OllamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type StreamChatInput struct {
	Message   string
	SessionID string
	Model     string
	APIKey    string
}

type StreamChatOutput struct {
	Stream     io.ReadCloser
	CommitFunc func(ctx context.Context, assistantContent string) error
}

type StreamChatUseCase struct {
	cache cache.Cache
}

func NewStreamChatUseCase(cache cache.Cache) *StreamChatUseCase {
	return &StreamChatUseCase{cache: cache}
}

func (uc *StreamChatUseCase) Execute(ctx context.Context, input StreamChatInput) (*StreamChatOutput, error) {
	apiKey := os.Getenv("CHAT_API_KEY")
	if apiKey == "" || input.APIKey != apiKey {
		return nil, ErrUnauthorized
	}

	history, err := uc.loadHistory(ctx, input.SessionID)
	if err != nil {
		history = []OllamaMessage{}
	}
	history = trimHistory(history, 3)

	messages := uc.buildMessages(history, input.Message)

	ollamaURL := fmt.Sprintf("%s/v1/chat/completions", os.Getenv("OLLAMA_URL"))
	reqBody, _ := json.Marshal(ollamaRequest{
		Model:    input.Model,
		Messages: messages,
		Stream:   true,
	})

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, ErrUpstreamUnavailable
	}
	upstreamReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		return nil, ErrUpstreamUnavailable
	}

	commitFunc := func(ctx context.Context, assistantContent string) error {
		if assistantContent == "" {
			return nil
		}
		updated := append(history,
			OllamaMessage{Role: "user", Content: input.Message},
			OllamaMessage{Role: "assistant", Content: assistantContent},
		)
		if err := uc.saveHistory(ctx, input.SessionID, trimHistory(updated, 3)); err != nil {
			log.Printf("ollama: failed to save history: %v", err)
		}
		return nil
	}

	return &StreamChatOutput{
		Stream:     resp.Body,
		CommitFunc: commitFunc,
	}, nil
}

func (uc *StreamChatUseCase) buildMessages(history []OllamaMessage, userMessage string) []OllamaMessage {
	var messages []OllamaMessage
	if systemPrompt := os.Getenv("OLLAMA_SYSTEM_PROMPT"); systemPrompt != "" {
		messages = append(messages, OllamaMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, history...)
	messages = append(messages, OllamaMessage{Role: "user", Content: userMessage})
	return messages
}

func (uc *StreamChatUseCase) loadHistory(ctx context.Context, sessionID string) ([]OllamaMessage, error) {
	data, ok, err := uc.cache.Get(ctx, "ollama:session:"+sessionID)
	if err != nil || !ok {
		return nil, err
	}
	var messages []OllamaMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (uc *StreamChatUseCase) saveHistory(ctx context.Context, sessionID string, messages []OllamaMessage) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return uc.cache.Set(ctx, "ollama:session:"+sessionID, data, historyTTL)
}

func trimHistory(messages []OllamaMessage, n int) []OllamaMessage {
	if len(messages) <= n {
		return messages
	}
	return messages[len(messages)-n:]
}
