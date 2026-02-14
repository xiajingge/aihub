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

	apiGroup := server.Group("/v1") //middleware.WithTimeout(server.Config.LLMRequestTimeout),
	//middleware.WithSource("api"),

	{
		apiGroup.POST("/chat/completions", h.OpenAI.ChatCompletion)
	}
	// TODO authropic
}
