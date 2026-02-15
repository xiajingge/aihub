package service

import (
	openaitransformer "github.com/xiajignge/aihub/internal/domain/transformer/openai"
	"github.com/xiajignge/aihub/internal/service/channel"
	"github.com/xiajignge/aihub/internal/service/executor"
	"github.com/xiajignge/aihub/internal/service/pipeline"
	"github.com/xiajignge/aihub/pkg/httpclient"
	"go.uber.org/fx"
)

const (
	OpenAIChatCompletionPipelineName = "openai_chat_completion_pipeline"
	OpenAIChatCompletionServiceName  = "openai_chat_completion_service"
)

var Module = fx.Module("service",
	channel.Module,
	openaitransformer.Module,
	fx.Provide(NewExecutor),
	fx.Provide(
		fx.Annotate(
			pipeline.NewPipelineWithDeps,
			fx.ParamTags(`name:"`+openaitransformer.InboundName+`"`, ``, ``, ``),
			fx.ResultTags(`name:"`+OpenAIChatCompletionPipelineName+`"`),
			fx.As(new(pipeline.Pipeline)),
		),
	),
	fx.Provide(
		fx.Annotate(
			NewChatCompletionProcessor,
			fx.ParamTags(`name:"`+OpenAIChatCompletionPipelineName+`"`),
			fx.ResultTags(`name:"`+OpenAIChatCompletionServiceName+`"`),
			fx.As(new(ChatCompletionService)),
		),
	),
)

func NewExecutor(client *httpclient.HttpClient) executor.Executor {
	return client
}
