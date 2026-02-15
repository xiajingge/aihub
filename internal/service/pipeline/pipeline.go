package pipeline

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
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
	p.logger.Debug("request received", logger.String("request_body", string(request.Body)))

	llmRequest, err := p.Inbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	channels, err := p.ChannelService.Select(ctx, llmRequest)
	if err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return nil, errors.New("channel not found")
	}

	var lastErr error
	for idx, ch := range channels {
		requestForChannel, err := buildRequestForChannel(llmRequest, ch)
		if err != nil {
			lastErr = err
			p.logger.Warn("skip channel due to unsupported model mapping",
				logger.Int("channel_index", idx),
				logger.String("channel", ch.Name),
				logger.Error(err),
			)
			continue
		}

		result, err := p.runWithOutbound(ctx, requestForChannel, ch.Outbound)
		if err == nil {
			if idx > 0 {
				p.logger.Info("request succeeded after fallback to next channel",
					logger.Int("channel_index", idx),
					logger.String("channel", ch.Name),
				)
			}
			return result, nil
		}

		lastErr = err
		p.logger.Warn("channel request failed",
			logger.Int("channel_index", idx),
			logger.String("channel", ch.Name),
			logger.Error(err),
		)

		if !shouldFallbackToNextChannel(err) {
			return nil, err
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all channels failed: %w", lastErr)
	}
	return nil, errors.New("channel not found")
}

func (p *pipeline) runWithOutbound(
	ctx context.Context,
	request *domain.Request,
	outbound transformer.Outbound,
) (*Result, error) {
	if request.Stream != nil && *request.Stream {
		resp, err := p.stream(ctx, request, outbound)
		if err != nil {
			return nil, err
		}
		return &Result{
			Stream:   true,
			Response: nil,
			SSEvent:  resp,
		}, nil
	}

	resp, err := p.notStream(ctx, request, outbound)
	if err != nil {
		return nil, err
	}
	return &Result{
		Stream:   false,
		Response: resp,
		SSEvent:  nil,
	}, nil
}

func buildRequestForChannel(req *domain.Request, ch *llm.Channel) (*domain.Request, error) {
	resolvedModel, err := ch.ChooseModel(req.Model)
	if err != nil {
		return nil, err
	}

	clonedReq := *req
	clonedReq.Model = resolvedModel
	return &clonedReq, nil
}

func shouldFallbackToNextChannel(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, transformer.ErrInvalidRequest) {
		return false
	}

	var httpErr *httpclient.Error
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden,
			http.StatusRequestTimeout, http.StatusTooManyRequests,
			http.StatusInternalServerError, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	msg := strings.ToLower(err.Error())
	retryableKeywords := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"broken pipe",
		"unexpected eof",
	}
	for _, keyword := range retryableKeywords {
		if strings.Contains(msg, keyword) {
			return true
		}
	}

	return false
}

func (p *pipeline) stream(ctx context.Context, request *domain.Request, outbound transformer.Outbound) (httpclient.Stream[*httpclient.StreamEvent], error) {
	httpReq, err := outbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	outboundStream, err := p.Executor.DoStream(ctx, httpReq)
	if err != nil {
		return nil, err
	}

	llmStream, err := outbound.TransformStream(ctx, outboundStream)
	if err != nil {
		p.logger.Error("failed to transform stream response to llm", logger.Error(err))
		return nil, err
	}

	inboundStream, err := p.Inbound.TransformStream(ctx, llmStream)
	if err != nil {
		p.logger.Error("failed to transform stream response to inbound format", logger.Error(err))
		return nil, err
	}

	return inboundStream, nil
}

func (p *pipeline) notStream(ctx context.Context, request *domain.Request, outbound transformer.Outbound) (*httpclient.Response, error) {
	httpReq, err := outbound.TransformRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	httpResp, err := p.Executor.Do(ctx, httpReq)
	if err != nil {
		return nil, err
	}

	llmResp, err := outbound.TransformResponse(ctx, httpResp)
	if err != nil {
		p.logger.Error("failed to transform non-stream response to llm", logger.Error(err))
		return nil, err
	}

	finalResp, err := p.Inbound.TransformResponse(ctx, llmResp)
	if err != nil {
		p.logger.Error("failed to transform non-stream response to inbound format", logger.Error(err))
		return nil, err
	}

	return finalResp, nil
}

type Result struct {
	Stream   bool
	Response *httpclient.Response
	SSEvent  httpclient.Stream[*httpclient.StreamEvent]
}
