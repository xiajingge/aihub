package channel

import (
	"context"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
	"github.com/xiajingge/logger"
)

type DefaultChannelSelector struct {
	ChannelService *ChannelService
	logger         logger.LoggerV1
}

func NewDefaultChannelSelector(channelService *ChannelService, logger logger.LoggerV1) *DefaultChannelSelector {
	return &DefaultChannelSelector{
		ChannelService: channelService,
		logger:         logger,
	}
}

func (s *DefaultChannelSelector) Select(ctx context.Context, req *domain.Request) ([]*llm.Channel, error) {
	// The request model has already been mapped by the inbound transformer if needed
	// Channel selection will use the mapped model for finding compatible channels
	channels, err := s.ChannelService.ChooseChannels(ctx, req)
	if err != nil {
		return nil, err
	}

	s.logger.Debug("Selected channels for model",
		logger.String("model", req.Model),
		logger.Int("channel_count", len(channels)))

	return channels, nil
}
