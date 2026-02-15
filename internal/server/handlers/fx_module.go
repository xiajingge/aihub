package handlers

import (
	openaihandlers "github.com/xiajignge/aihub/internal/server/handlers/openai"
	"go.uber.org/fx"
)

var Module = fx.Module("handlers",
	openaihandlers.Module,
)
