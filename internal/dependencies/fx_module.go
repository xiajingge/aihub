package dependencies

import (
	"go.uber.org/fx"
)

var Module = fx.Module("dependencies",
	fx.Provide(NewLogger),
	fx.Provide(NewEntClient),
	fx.Provide(NewHttpClient),
	fx.Provide(NewErrorHandler, NewExecutors),
)

var DependencyModule = Module
