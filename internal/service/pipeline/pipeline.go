package pipeline

import (
	"context"
	"time"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/transformer"
	"github.com/xiajignge/aihub/internal/service/channel"
	"github.com/xiajignge/aihub/internal/service/executor"
	"github.com/xiajignge/aihub/pkg/httpclient"
	"github.com/xiajingge/logger"
	"go.uber.org/fx"
)

type pipeline struct {
	Inbound        transformer.Inbound
	ChannelService channel.ChannelSelector
	Executor       executor.Executor
	logger         logger.LoggerV1

	maxRetries      int
	retryDelay      time.Duration
	retryableErrors []string
}

type pipelineParams struct {
	fx.In

	Inbound         transformer.Inbound
	ChannelSelector channel.ChannelSelector
	Executor        executor.Executor
	logger          logger.LoggerV1
}
type Option func(*pipeline)

func WithMaxRetries(maxRetries int) Option {
	return func(p *pipeline) {
		p.maxRetries = maxRetries
	}
}

func NewPipelineWithDeps(
	inbound transformer.Inbound,
	channelSelector channel.ChannelSelector,
	exec executor.Executor,
	log logger.LoggerV1,
	opts ...Option,
) *pipeline {
	p := &pipeline{
		Inbound:        inbound,
		ChannelService: channelSelector,
		Executor:       exec,
		logger:         log,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func NewPipeline(params pipelineParams, opts ...Option) *pipeline {
	return NewPipelineWithDeps(
		params.Inbound,
		params.ChannelSelector,
		params.Executor,
		params.logger,
		opts...,
	)
}
func (p *pipeline) Run(ctx context.Context, request *httpclient.Request) (*Result, error) {
	// 记录请求日志（含请求体）
	p.logger.Debug("request received", logger.String("request_body", string(request.Body)))

	// 开始处理请求
	llmRequest, err := p.Inbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	// 根据inbound来选择outbound
	channels, err := p.ChannelService.SelectOne(ctx, llmRequest)
	if err != nil {
		return nil, err
	}

	outbound := channels.Outbound

	if *llmRequest.Stream {
		resp, err := p.stream(ctx, llmRequest, outbound)
		if err != nil {
			return nil, err
		}
		return &Result{
			Stream:   true,
			Response: nil,
			SSEvent:  resp,
		}, nil
	} else {
		resp, err := p.notStream(ctx, llmRequest, outbound)
		if err != nil {
			return nil, err
		}
		return &Result{
			Stream:   true,
			Response: resp,
			SSEvent:  nil,
		}, nil
	}
}

func (p *pipeline) stream(ctx context.Context, request *domain.Request, outbound transformer.Outbound) (httpclient.Stream[*httpclient.StreamEvent], error) {
	// 1.将内部的请求转换成特定提供商的请求格式
	httpReq, err := outbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	// 2.执行stream http请求
	outboundStream, err := p.Executor.DoStream(ctx, httpReq)

	if err != nil {
		return nil, err
	}

	// 3. 将特定提供商返回的信息转换为内部响应体
	llmStream, err := outbound.TransformStream(ctx, outboundStream)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	// 4. 将内部响应体转换为用户请求对应的响应格式
	inboundStream, err := p.Inbound.TransformStream(ctx, llmStream)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	return inboundStream, nil
}

func (p *pipeline) notStream(ctx context.Context, request *domain.Request, outbound transformer.Outbound) (*httpclient.Response, error) {
	// 1.将内部的请求转换成特定提供商的请求格式
	httpReq, err := outbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	// 2.执行 http 请求
	httpResp, err := p.Executor.Do(ctx, httpReq)

	if err != nil {
		return nil, err
	}

	// 3. 将特定提供商返回的信息转换为内部响应体
	llmResp, err := outbound.TransformResponse(ctx, httpResp)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	// 4. 将内部响应体转换为用户请求对应的响应格式
	finalResp, err := p.Inbound.TransformResponse(ctx, llmResp)
	if err != nil {
		p.logger.Error("Failed to transform streaming request", logger.Error(err))
		return nil, err
	}

	return finalResp, nil
}

type Result struct {
	Stream   bool
	Response *httpclient.Response
	SSEvent  httpclient.Stream[*httpclient.StreamEvent]
}
