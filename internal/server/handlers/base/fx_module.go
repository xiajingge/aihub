package base

import (
	"github.com/xiajignge/aihub/internal/service"
	"github.com/xiajingge/logger"
	"go.uber.org/fx"
)

var Module = fx.Module("base_handlers",
	fx.Provide(NewChatCompletionSSEHandlers),
)

func NewChatCompletionSSEHandlers(
	chatCompletionProcessor *service.ChatCompletionProcessor,
	loggerv1 logger.LoggerV1,
) *ChatCompletionSSEHandlers {
	return &ChatCompletionSSEHandlers{
		ChatCompletionProcessor: chatCompletionProcessor,
		logger:                  loggerv1,
	}
}
