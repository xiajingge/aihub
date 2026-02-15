package openai

import (
	"github.com/xiajignge/aihub/internal/server/handlers/base"
	"github.com/xiajignge/aihub/internal/service"
	"go.uber.org/fx"
)

const (
	chatCompletionHandlerName = "openai_chat_completion_handler"
)

var Module = fx.Module("openai_handlers",
	fx.Provide(
		fx.Annotate(
			base.NewChatCompletionSSEHandlers,
			fx.ParamTags(`name:"`+service.OpenAIChatCompletionServiceName+`"`, ``),
			fx.ResultTags(`name:"`+chatCompletionHandlerName+`"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewOpenAIHandlers,
			fx.ParamTags(`name:"`+chatCompletionHandlerName+`"`),
		),
	),
)
