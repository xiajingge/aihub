package channel

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
	"github.com/xiajingge/logger"
)

type DefaultChannelSelector struct {
	ChannelService *ChannelService
	logger         logger.LoggerV1
	roundRobinSeq  atomic.Uint64
}

const (
	channelSelectStrategyMetadataKey = "channel_select_strategy"
	channelSelectStrategyRoundRobin  = "round_robin"
	channelSelectStrategyMaxWeight   = "max_weight"
)

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

func (s *DefaultChannelSelector) SelectOne(ctx context.Context, req *domain.Request) (*llm.Channel, error) {
	// The request model has already been mapped by the inbound transformer if needed
	// Channel selection will use the mapped model for finding compatible channels
	channels, err := s.ChannelService.ChooseChannels(ctx, req)
	if err != nil {
		return nil, err
	}

	s.logger.Debug("Selected channels for model",
		logger.String("model", req.Model),
		logger.Int("channel_count", len(channels)))

	if len(channels) == 0 {
		return nil, errors.New("channel not found")
	}

	strategy := resolveSelectOneStrategy(req)
	selected, selectedIdx := s.pickOneByRoundRobin(channels)
	if strategy == channelSelectStrategyMaxWeight {
		selected, selectedIdx = s.pickOneByMaxWeight(channels)
	}

	s.logger.Debug("Selected one channel for model",
		logger.String("model", req.Model),
		logger.String("strategy", strategy),
		logger.Int("selected_index", selectedIdx),
		logger.String("selected_channel", selected.Name))

	return selected, nil
}

func (s *DefaultChannelSelector) pickOneByRoundRobin(channels []*llm.Channel) (*llm.Channel, int) {
	if len(channels) == 1 {
		return channels[0], 0
	}

	seq := s.roundRobinSeq.Add(1) - 1
	idx := int(seq % uint64(len(channels)))
	return channels[idx], idx
}

func (s *DefaultChannelSelector) pickOneByMaxWeight(channels []*llm.Channel) (*llm.Channel, int) {
	maxIdx := 0
	maxWeight := channels[0].OrderingWeight

	for i := 1; i < len(channels); i++ {
		if channels[i].OrderingWeight > maxWeight {
			maxWeight = channels[i].OrderingWeight
			maxIdx = i
		}
	}

	return channels[maxIdx], maxIdx
}

func resolveSelectOneStrategy(req *domain.Request) string {
	if req == nil || req.Metadata == nil {
		return channelSelectStrategyRoundRobin
	}

	strategy := strings.TrimSpace(strings.ToLower(req.Metadata[channelSelectStrategyMetadataKey]))
	switch strategy {
	case channelSelectStrategyMaxWeight:
		return channelSelectStrategyMaxWeight
	default:
		return channelSelectStrategyRoundRobin
	}
}
