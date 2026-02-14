package openai

import (
	"github.com/xiajignge/aihub/internal/domain/transformer"
	"go.uber.org/fx"
)

var Module = fx.Module("openai_transformer",
	fx.Provide(
		fx.Annotate(
			NewInboundTransformer,
			fx.As(new(transformer.Inbound)),
		),
	),
)
