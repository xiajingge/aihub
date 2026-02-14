package openai

import (
	"github.com/xiajignge/aihub/internal/server/handlers/base"
	"go.uber.org/fx"
)

var Module = fx.Module("openai_handlers",
	fx.Provide(NewOpenAIHandlers),
)

func NewOpenAIHandlers(chatCompletionHandlers *base.ChatCompletionSSEHandlers) *OpenAIHandlers {
	return &OpenAIHandlers{
		ChatCompletionHandlers: chatCompletionHandlers,
	}
}
