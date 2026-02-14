package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xiajignge/aihub/config"
	"github.com/xiajingge/logger"
)

type Server struct {
	*gin.Engine
	Config   *config.ServerConfig
	server   *http.Server
	addr     string
	loggerv1 logger.LoggerV1
}

func NewServer(engine *gin.Engine, cfg config.ServerConfig, loggerv1 logger.LoggerV1) *Server {
	return &Server{Engine: engine, Config: &cfg, loggerv1: loggerv1}
}

func (srv *Server) Run() error {
	srv.addr = fmt.Sprintf("0.0.0.0:%d", srv.Config.Port)

	srv.loggerv1.Info(
		"run server",
		logger.String("server.name",
			srv.Config.Name),
		logger.String("server.addr", srv.addr),
	)

	srv.server = &http.Server{
		Addr:         srv.addr,
		Handler:      srv.Engine,
		ReadTimeout:  srv.Config.ReadTimeout,
		WriteTimeout: srv.Config.WriteTimeout,
	}

	err := srv.server.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	}

	return nil
}

func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.server.Shutdown(ctx)
}
