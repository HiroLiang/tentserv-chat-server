package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/cache"
	"github.com/gin-gonic/gin"
)

const historyTTL = 24 * time.Hour

type OllamaChatHandler struct {
	cache cache.Cache
}

func NewOllamaChatHandler(cache cache.Cache) *OllamaChatHandler {
	return &OllamaChatHandler{cache: cache}
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func trimHistory(messages []ollamaMessage, n int) []ollamaMessage {
	if len(messages) <= n {
		return messages
	}
	return messages[len(messages)-n:]
}

func (h *OllamaChatHandler) Stream(c *gin.Context) {
	apiKey := os.Getenv("CHAT_API_KEY")
	if apiKey == "" || c.GetHeader("X-Chat-Api-Key") != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	message := c.Query("message")
	sessionID := c.Query("session_id")
	model := c.Query("model")

	if message == "" || sessionID == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message, session_id, and model are required"})
		return
	}

	ctx := c.Request.Context()

	history, err := h.loadHistory(ctx, sessionID)
	if err != nil {
		history = []ollamaMessage{}
	}
	history = trimHistory(history, 3)

	var messages []ollamaMessage
	if systemPrompt := os.Getenv("OLLAMA_SYSTEM_PROMPT"); systemPrompt != "" {
		messages = append(messages, ollamaMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, history...)
	messages = append(messages, ollamaMessage{Role: "user", Content: message})

	ollamaURL := fmt.Sprintf("%s/v1/chat/completions", os.Getenv("OLLAMA_URL"))
	reqBody, _ := json.Marshal(ollamaRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	})

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaURL, bytes.NewReader(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach upstream"})
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("ollama: failed to close response body: %v", err)
		}
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	var assistantContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if _, err := fmt.Fprint(c.Writer, "\n"); err != nil {
				break
			}
			c.Writer.Flush()
			continue
		}

		if _, err := fmt.Fprintf(c.Writer, "%s\n", line); err != nil {
			break
		}
		c.Writer.Flush()

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				continue
			}
			var chunk ollamaChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				for _, choice := range chunk.Choices {
					assistantContent.WriteString(choice.Delta.Content)
				}
			}
		}
	}

	if assistantContent.Len() > 0 {
		history = append(history, ollamaMessage{Role: "user", Content: message})
		history = append(history, ollamaMessage{Role: "assistant", Content: assistantContent.String()})
		_ = h.saveHistory(ctx, sessionID, trimHistory(history, 3))
	}
}

func (h *OllamaChatHandler) loadHistory(ctx context.Context, sessionID string) ([]ollamaMessage, error) {
	data, ok, err := h.cache.Get(ctx, "ollama:session:"+sessionID)
	if err != nil || !ok {
		return nil, err
	}
	var messages []ollamaMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func (h *OllamaChatHandler) saveHistory(ctx context.Context, sessionID string, messages []ollamaMessage) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return h.cache.Set(ctx, "ollama:session:"+sessionID, data, historyTTL)
}
