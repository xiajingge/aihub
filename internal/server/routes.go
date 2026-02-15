package server

import (
	"github.com/xiajignge/aihub/internal/server/handlers/openai"
	"go.uber.org/fx"
)

type Handlers struct {
	fx.In

	OpenAI *openai.OpenAIHandlers
}

func (h *Handlers) SetupRoutes(server *Server) {
	// TODO add middleware
	// TODO noroute \ tracing\ metric \ jwt

	h.setupOpenAIRoutes(server)
	// TODO: h.setupAnthropicRoutes(server)
	// TODO: h.setupGoogleRoutes(server)
}

func (h *Handlers) setupOpenAIRoutes(server *Server) {
	apiGroup := server.Group("/v1")
	apiGroup.POST("/chat/completions", h.OpenAI.ChatCompletion)
}
