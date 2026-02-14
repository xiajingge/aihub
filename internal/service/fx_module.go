package service

import (
	openaitransformer "github.com/xiajignge/aihub/internal/domain/transformer/openai"
	"github.com/xiajignge/aihub/internal/service/channel"
	"github.com/xiajignge/aihub/internal/service/executor"
	"github.com/xiajignge/aihub/pkg/httpclient"
	"go.uber.org/fx"
)

var Module = fx.Module("service",
	channel.Module,
	openaitransformer.Module,
	fx.Provide(NewExecutor),
	fx.Provide(NewChatCompletionProcessor),
)

func NewExecutor(client *httpclient.HttpClient) executor.Executor {
	return client
}
