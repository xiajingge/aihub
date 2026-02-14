package openai

import (
	"github.com/gin-gonic/gin"
	"github.com/xiajignge/aihub/internal/server/handlers/base"
)

type OpenAIHandlers struct {
	ChatCompletionHandlers *base.ChatCompletionSSEHandlers
}

func (handlers *OpenAIHandlers) ChatCompletion(c *gin.Context) {
	handlers.ChatCompletionHandlers.ChatCompletion(c)
}
