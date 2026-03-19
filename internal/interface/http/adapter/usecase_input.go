package adapter

import (
	"net/http"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

func MustBaseInput(c *gin.Context) *shared.BaseContext {
	metadata, ok := c.Get("context")
	if !ok {
		c.JSON(http.StatusInternalServerError, response.ErrNotFound("metadata"))
	}
	return metadata.(*shared.BaseContext)
}

func BuildInput[T any](c *gin.Context, data T) shared.UseCaseInput[T] {
	return shared.UseCaseInput[T]{
		Base: *MustBaseInput(c),
		Data: data,
	}
}

func BuildEmptyInput(c *gin.Context) shared.UseCaseInput[struct{}] {
	return shared.UseCaseInput[struct{}]{
		Base: *MustBaseInput(c),
	}
}
