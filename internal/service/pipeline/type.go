package pipeline

import (
	"context"

	"github.com/xiajignge/aihub/pkg/httpclient"
)

type Pipeline interface {
	Run(ctx context.Context, request *httpclient.Request) (*Result, error)
}
