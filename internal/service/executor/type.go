package executor

import (
	"context"

	"github.com/xiajignge/aihub/pkg/httpclient"
)

type Executor interface {
	Do(ctx context.Context, request *httpclient.Request) (*httpclient.Response, error)
	DoStream(ctx context.Context, request *httpclient.Request) (httpclient.Stream[*httpclient.StreamEvent], error)
}
