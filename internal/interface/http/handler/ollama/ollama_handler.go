package ollama

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	ollamaUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/ollama/usecase"
	"github.com/gin-gonic/gin"
)

type ollamaChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type OllamaChatHandler struct {
	streamChatUseCase *ollamaUseCase.StreamChatUseCase
}

func NewOllamaChatHandler(uc *ollamaUseCase.StreamChatUseCase) *OllamaChatHandler {
	return &OllamaChatHandler{streamChatUseCase: uc}
}

func (h *OllamaChatHandler) Stream(c *gin.Context) {
	message := c.Query("message")
	sessionID := c.Query("session_id")
	model := c.Query("model")

	if message == "" || sessionID == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message, session_id, and model are required"})
		return
	}

	output, err := h.streamChatUseCase.Execute(c.Request.Context(), ollamaUseCase.StreamChatInput{
		Message:   message,
		SessionID: sessionID,
		Model:     model,
		APIKey:    c.GetHeader("X-Chat-Api-Key"),
	})
	if err != nil {
		switch err {
		case ollamaUseCase.ErrUnauthorized:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		default:
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach upstream"})
		}
		return
	}
	defer func() {
		if err := output.Stream.Close(); err != nil {
			log.Printf("ollama: failed to close response body: %v", err)
		}
	}()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	var assistantContent strings.Builder
	scanner := bufio.NewScanner(output.Stream)
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

	_ = output.CommitFunc(c.Request.Context(), assistantContent.String())
}
