package channel

import "go.uber.org/fx"

var Module = fx.Module("channel_service",
	fx.Provide(NewChannelService),
	fx.Provide(
		fx.Annotate(
			NewDefaultChannelSelector,
			fx.As(new(ChannelSelector)),
		),
	),
)

var channelService = Module
