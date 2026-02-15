package channel

import (
	"context"
	"fmt"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
	"github.com/xiajignge/aihub/internal/objects"
)

type SpecifiedChannelSelector struct {
	ChannelService *ChannelService
	ChannelID      objects.GUID
}

func NewSpecifiedChannelSelector(channelService *ChannelService, channelID objects.GUID) *SpecifiedChannelSelector {
	return &SpecifiedChannelSelector{
		ChannelService: channelService,
		ChannelID:      channelID,
	}
}

func (s *SpecifiedChannelSelector) Select(ctx context.Context, req *domain.Request) ([]*llm.Channel, error) {
	channel, err := s.ChannelService.GetChannelForTest(ctx, s.ChannelID.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel for test: %w", err)
	}

	if !channel.IsModelSupported(req.Model) {
		return nil, fmt.Errorf("model %s not supported in channel %s", req.Model, channel.Name)
	}

	return []*llm.Channel{channel}, nil
}

func (s *SpecifiedChannelSelector) SelectOne(ctx context.Context, req *domain.Request) (*llm.Channel, error) {
	channels, err := s.Select(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(channels) == 0 {
		return nil, fmt.Errorf("channel %d not found", s.ChannelID.ID)
	}

	return channels[0], nil
}
