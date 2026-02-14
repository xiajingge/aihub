package handlers

import (
	"github.com/xiajignge/aihub/internal/server/handlers/base"
	openaihandlers "github.com/xiajignge/aihub/internal/server/handlers/openai"
	"go.uber.org/fx"
)

var Module = fx.Module("handlers",
	base.Module,
	openaihandlers.Module,
)
