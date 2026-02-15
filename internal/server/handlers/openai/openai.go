package openai

import (
	"github.com/gin-gonic/gin"
	"github.com/xiajignge/aihub/internal/server/handlers/base"
)

type OpenAIHandlers struct {
	ChatCompletionHandlers *base.ChatCompletionSSEHandlers
}

func NewOpenAIHandlers(chatCompletionHandlers *base.ChatCompletionSSEHandlers) *OpenAIHandlers {
	return &OpenAIHandlers{
		ChatCompletionHandlers: chatCompletionHandlers,
	}
}

func (handlers *OpenAIHandlers) ChatCompletion(c *gin.Context) {
	handlers.ChatCompletionHandlers.ChatCompletion(c)
}
