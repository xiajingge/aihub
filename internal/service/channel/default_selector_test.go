package channel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xiajignge/aihub/internal/domain"
	"github.com/xiajignge/aihub/internal/domain/llm"
	"github.com/xiajignge/aihub/internal/ent"
	"github.com/xiajingge/logger"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...logger.Field) {}
func (noopLogger) Info(string, ...logger.Field)  {}
func (noopLogger) Warn(string, ...logger.Field)  {}
func (noopLogger) Error(string, ...logger.Field) {}

func TestDefaultChannelSelectorSelectOneRoundRobin(t *testing.T) {
	selector := NewDefaultChannelSelector(
		&ChannelService{
			Channels: []*llm.Channel{
				newWeightedTestChannel("channel-a", 10, "gpt-4o"),
				newWeightedTestChannel("channel-b", 8, "gpt-4o"),
				newWeightedTestChannel("channel-c", 6, "gpt-4o"),
			},
		},
		noopLogger{},
	)

	req := &domain.Request{Model: "gpt-4o"}
	expected := []string{"channel-a", "channel-b", "channel-c", "channel-a", "channel-b"}

	for _, channelName := range expected {
		ch, err := selector.SelectOne(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, channelName, ch.Name)
	}
}

func TestDefaultChannelSelectorSelectOneOnlyRotatesSupportedChannels(t *testing.T) {
	selector := NewDefaultChannelSelector(
		&ChannelService{
			Channels: []*llm.Channel{
				newWeightedTestChannel("channel-a", 10, "gpt-4o"),
				newWeightedTestChannel("channel-b", 9, "o3-mini"),
				newWeightedTestChannel("channel-c", 8, "gpt-4o"),
			},
		},
		noopLogger{},
	)

	req := &domain.Request{Model: "gpt-4o"}
	expected := []string{"channel-a", "channel-c", "channel-a", "channel-c"}

	for _, channelName := range expected {
		ch, err := selector.SelectOne(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, channelName, ch.Name)
	}
}

func TestDefaultChannelSelectorSelectOneReturnsErrorWhenNotFound(t *testing.T) {
	selector := NewDefaultChannelSelector(
		&ChannelService{
			Channels: []*llm.Channel{
				newWeightedTestChannel("channel-a", 10, "gpt-4o"),
			},
		},
		noopLogger{},
	)

	_, err := selector.SelectOne(context.Background(), &domain.Request{Model: "not-exists"})
	require.EqualError(t, err, "channel not found")
}

func TestDefaultChannelSelectorSelectOneMaxWeight(t *testing.T) {
	selector := NewDefaultChannelSelector(
		&ChannelService{
			Channels: []*llm.Channel{
				newWeightedTestChannel("channel-a", 10, "gpt-4o"),
				newWeightedTestChannel("channel-b", 100, "gpt-4o"),
				newWeightedTestChannel("channel-c", 50, "gpt-4o"),
			},
		},
		noopLogger{},
	)

	req := &domain.Request{
		Model: "gpt-4o",
		Metadata: map[string]string{
			channelSelectStrategyMetadataKey: channelSelectStrategyMaxWeight,
		},
	}

	for range 5 {
		ch, err := selector.SelectOne(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, "channel-b", ch.Name)
	}
}

func newWeightedTestChannel(name string, weight int, models ...string) *llm.Channel {
	return &llm.Channel{
		Channel: &ent.Channel{
			Name:            name,
			OrderingWeight:  weight,
			SupportedModels: models,
		},
	}
}
