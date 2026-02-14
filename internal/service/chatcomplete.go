package service

import (
	"context"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/transformer"
	"github.com/xiajignge/aihub/internal/service/channel"
	"github.com/xiajignge/aihub/internal/service/executor"
	"github.com/xiajignge/aihub/pkg/httpclient"
	"github.com/xiajingge/logger"
)

type ContextKey string

const (
	// APIKeyContextKey 用于在 context 中存储 API key entity.
	APIKeyContextKey ContextKey = "api_key"
	// UserContextKey 用于在 context 中存储用户 entity.
	UserContextKey ContextKey = "user"
)

type ChatCompletionProcessor struct {
	// ChannelSelector 负责选择可用通道的策略。
	ChannelSelector channel.ChannelSelector
	// Inbound 负责将请求转换为内部统一格式的入站转换器。
	Inbound transformer.Inbound
	// Executor 负责网络发送
	Executor executor.Executor

	// Logger 日志功能
	logger logger.LoggerV1
	// TODO
}

func NewChatCompletionProcessor(channelSelector channel.ChannelSelector, inbound transformer.Inbound, executor executor.Executor, logger logger.LoggerV1) *ChatCompletionProcessor {
	return &ChatCompletionProcessor{ChannelSelector: channelSelector, Inbound: inbound, Executor: executor, logger: logger}
}

type ChatCompletionResult struct {
	ChatCompletion       *httpclient.Response
	ChatCompletionStream httpclient.Stream[*httpclient.StreamEvent]
}

func (p *ChatCompletionProcessor) Process(ctx context.Context, request *httpclient.Request) (*ChatCompletionResult, error) {
	// todo 从上下文中获取 API Key 与用户信息
	// apiKey, _ := ctx.Value(APIKeyContextKey).(*ent.APIKey)
	// user, _ := ctx.Value(UserContextKey).(*ent.User)

	// 记录请求日志（含请求体）
	p.logger.Debug("request received", logger.String("request_body", string(request.Body)))

	// 开始处理请求
	llmRequest, err := p.Inbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	return p.processRequest(ctx, llmRequest)

}

func (p *ChatCompletionProcessor) processRequest(ctx context.Context, req *domain.Request) (*ChatCompletionResult, error) {
	if *req.Stream {
		resp, err := p.stream(ctx, req)
		if err != nil {
			return nil, err
		}
		return &ChatCompletionResult{
			ChatCompletion:       nil,
			ChatCompletionStream: resp,
		}, nil
	} else {
		resp, err := p.notStream(ctx, req)
		if err != nil {
			return nil, err
		}
		return &ChatCompletionResult{
			ChatCompletion:       resp,
			ChatCompletionStream: nil,
		}, nil
	}

}

func (p *ChatCompletionProcessor) stream(ctx context.Context, request *domain.Request) (httpclient.Stream[*httpclient.StreamEvent], error) {
	// 1.选择channel
	channels, err := p.ChannelSelector.Select(ctx, request)
	if err != nil {
		return nil, err
	}
	// TODO负载均衡选择channel
	channel := channels[0]
	model, err := channel.ChooseModel(request.Model)
	if err != nil {
		p.logger.Error("Failed to choose model", logger.Error(err))
		return nil, err
	}
	request.Model = model

	// 2.将内部的请求转换成特定提供商的请求格式
	httpReq, err := channel.Outbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	// 3.执行stream http请求
	outboundStream, err := p.Executor.DoStream(ctx, httpReq)

	if err != nil {
		return nil, err
	}

	// 4. 将特定提供商返回的信息转换为内部响应体
	llmStream, err := channel.Outbound.TransformStream(ctx, outboundStream)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	// 5. 将内部响应体转换为用户请求对应的响应格式
	inboundStream, err := p.Inbound.TransformStream(ctx, llmStream)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	return inboundStream, nil
}

func (p *ChatCompletionProcessor) notStream(
	ctx context.Context,
	request *domain.Request,
) (*httpclient.Response, error) {
	// 1.选择channel
	channels, err := p.ChannelSelector.Select(ctx, request)
	// TODO负载均衡选择channel
	channel := channels[0]
	model, err := channel.ChooseModel(request.Model)
	if err != nil {
		p.logger.Error("Failed to choose model", logger.Error(err))
		return nil, err
	}
	request.Model = model

	// 2.将内部的请求转换成特定提供商的请求格式
	httpReq, err := channel.Outbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	// 3.执行 http 请求
	httpResp, err := p.Executor.Do(ctx, httpReq)

	if err != nil {
		return nil, err
	}

	// 4. 将特定提供商返回的信息转换为内部响应体
	llmResp, err := channel.Outbound.TransformResponse(ctx, httpResp)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	// 5. 将内部响应体转换为用户请求对应的响应格式
	finalResp, err := p.Inbound.TransformResponse(ctx, llmResp)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	return finalResp, nil
}
