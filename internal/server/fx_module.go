package server

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xiajignge/aihub/config"
	"github.com/xiajignge/aihub/internal/server/handlers"
	"github.com/xiajingge/logger"
	"go.uber.org/fx"
)

var Module = fx.Module("server",
	handlers.Module,
	fx.Provide(NewGinEngine),
	fx.Provide(NewServer),
	fx.Invoke(RegisterRoutes),
	fx.Invoke(RegisterLifecycle),
)

func NewGinEngine(cfg config.ServerConfig) *gin.Engine {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	return engine
}

func RegisterRoutes(h Handlers, server *Server) {
	h.SetupRoutes(server)
}

func RegisterLifecycle(lifecycle fx.Lifecycle, server *Server, loggerv1 logger.LoggerV1) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				err := server.Run()
				if err != nil {
					loggerv1.Error("server run error", logger.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}
