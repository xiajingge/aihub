package channel

import (
	"context"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
)

type ChannelSelector interface {
	Select(ctx context.Context, req *domain.Request) ([]*llm.Channel, error)
}
