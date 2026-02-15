package service

import (
	"context"

	"github.com/xiajignge/aihub/internal/service/pipeline"
	"github.com/xiajignge/aihub/pkg/httpclient"
)

type ContextKey string

const (
	APIKeyContextKey ContextKey = "api_key"
	UserContextKey   ContextKey = "user"
)

type ChatCompletionService interface {
	Process(ctx context.Context, request *httpclient.Request) (*ChatCompletionResult, error)
}

type ChatCompletionProcessor struct {
	pipeline pipeline.Pipeline
}

func NewChatCompletionProcessor(pl pipeline.Pipeline) *ChatCompletionProcessor {
	return &ChatCompletionProcessor{pipeline: pl}
}

type ChatCompletionResult struct {
	ChatCompletion       *httpclient.Response
	ChatCompletionStream httpclient.Stream[*httpclient.StreamEvent]
}

func (p *ChatCompletionProcessor) Process(ctx context.Context, request *httpclient.Request) (*ChatCompletionResult, error) {
	result, err := p.pipeline.Run(ctx, request)
	if err != nil {
		return nil, err
	}

	return &ChatCompletionResult{
		ChatCompletion:       result.Response,
		ChatCompletionStream: result.SSEvent,
	}, nil
}
