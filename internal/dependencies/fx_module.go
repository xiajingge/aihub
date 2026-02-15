package dependencies

import (
	"context"

	"github.com/xiajignge/aihub/internal/ent"
	"go.uber.org/fx"
)

var Module = fx.Module("dependencies",
	fx.Provide(NewLogger),
	fx.Provide(NewEntClient),
	fx.Provide(NewHttpClient),
	fx.Provide(NewErrorHandler, NewExecutors),
	fx.Invoke(func(lc fx.Lifecycle, client *ent.Client) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return client.Close()
			},
		})
	}),
)

var DependencyModule = Module
