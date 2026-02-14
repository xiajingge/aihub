package transformer

import (
	"context"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/pkg/httpclient"
)

// Inbound represents a transformer accpet the request from user and respond to use the transformed response.
// e.g: OpenAPI transformer accepts the request from user with OpenAPI format and respond with OpenAI format.
type Inbound interface {
	// APIFormat returns the API format of the transformer.
	APIFormat() domain.APIFormat

	// TransformRequest 将外部请求 -> 内部api
	TransformRequest(ctx context.Context, request *httpclient.Request) (*domain.Request, error)

	// TransformResponse 内部非流式响应 -> 外部非流式响应
	TransformResponse(ctx context.Context, response *domain.Response) (*httpclient.Response, error)

	// TransformStream 内部流式响应 -> 外部流式响应
	TransformStream(ctx context.Context, stream httpclient.Stream[*domain.Response]) (httpclient.Stream[*httpclient.StreamEvent], error)

	// TransformError 内部错误 -> 外部错误
	TransformError(ctx context.Context, err error) *httpclient.Error

	// AggregateStreamChunks 将流式响应组装成完整的响应
	AggregateStreamChunks(ctx context.Context, chunks []*httpclient.StreamEvent) ([]byte, domain.ResponseMeta, error)
}

// Outbound represents a transformer that convert the generic Request to the undering provider format.
// And transform the response from the undering provider format to generic Response format.
type Outbound interface {
	// APIFormat 标识当前供应商
	// e.g: openai/chat_completions, claude/messages.
	APIFormat() domain.APIFormat

	// TransformError 供应商错误 -> 内部错误
	TransformError(ctx context.Context, err *httpclient.Error) *domain.ResponseError

	// TransformRequest 内部api -> 供应商api
	TransformRequest(ctx context.Context, request *domain.Request) (*httpclient.Request, error)

	// TransformResponse 供应商的非流式响应 -> 内部api
	TransformResponse(ctx context.Context, response *httpclient.Response) (*domain.Response, error)

	// TransformStream 供应商的流式响应 -> 内部api
	TransformStream(ctx context.Context, stream httpclient.Stream[*httpclient.StreamEvent]) (httpclient.Stream[*domain.Response], error)

	// AggregateStreamChunks 流结束时做聚合
	AggregateStreamChunks(ctx context.Context, chunks []*httpclient.StreamEvent) ([]byte, domain.ResponseMeta, error)
}
