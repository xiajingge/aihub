package pipeline

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xiajignge/aihub/pkg/httpclient"
	"github.com/xiajingge/logger"
)

// RetryOption 配置可重试管道装饰器。
type RetryOption func(*RetryPipeline)

// RetryPipeline 在基础 Pipeline 外层增加可重试能力。
type RetryPipeline struct {
	next                     Pipeline
	logger                   logger.LoggerV1
	maxRetries               int
	retryDelay               time.Duration
	retryableStatusCodes     map[int]struct{}
	retryableErrorSubstrings []string
}

// NewRetryPipelineDecorator 创建带重试能力的 Pipeline 装饰器。
func NewRetryPipelineDecorator(next Pipeline, log logger.LoggerV1, opts ...RetryOption) Pipeline {
	p := &RetryPipeline{
		next:       next,
		logger:     log,
		maxRetries: 2,
		retryDelay: 200 * time.Millisecond,
		retryableStatusCodes: map[int]struct{}{
			http.StatusRequestTimeout:      {},
			http.StatusTooManyRequests:     {},
			http.StatusInternalServerError: {},
			http.StatusBadGateway:          {},
			http.StatusServiceUnavailable:  {},
			http.StatusGatewayTimeout:      {},
		},
		retryableErrorSubstrings: []string{
			"timeout",
			"connection reset",
			"connection refused",
			"temporarily unavailable",
			"broken pipe",
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *RetryPipeline) Run(ctx context.Context, request *httpclient.Request) (*Result, error) {
	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, err := p.next.Run(ctx, request)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !p.shouldRetry(err) || attempt == p.maxRetries {
			return nil, lastErr
		}

		p.logger.Warn("pipeline run failed, retrying",
			logger.Int("attempt", attempt+1),
			logger.Int("max_retries", p.maxRetries),
			logger.Error(err),
		)

		if err := p.waitRetryDelay(ctx); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func (p *RetryPipeline) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var httpErr *httpclient.Error
	if errors.As(err, &httpErr) {
		_, ok := p.retryableStatusCodes[httpErr.StatusCode]
		if ok {
			return true
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	errLower := strings.ToLower(err.Error())
	for _, keyword := range p.retryableErrorSubstrings {
		if strings.Contains(errLower, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

func (p *RetryPipeline) waitRetryDelay(ctx context.Context) error {
	if p.retryDelay <= 0 {
		return nil
	}

	timer := time.NewTimer(p.retryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func WithRetryMaxRetries(maxRetries int) RetryOption {
	return func(p *RetryPipeline) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		p.maxRetries = maxRetries
	}
}

func WithRetryDelay(delay time.Duration) RetryOption {
	return func(p *RetryPipeline) {
		p.retryDelay = delay
	}
}

func WithRetryableStatusCodes(statusCodes ...int) RetryOption {
	return func(p *RetryPipeline) {
		m := make(map[int]struct{}, len(statusCodes))
		for _, code := range statusCodes {
			m[code] = struct{}{}
		}
		p.retryableStatusCodes = m
	}
}

func WithRetryableErrorSubstrings(keywords ...string) RetryOption {
	return func(p *RetryPipeline) {
		p.retryableErrorSubstrings = append([]string(nil), keywords...)
	}
}
