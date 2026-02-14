package main

import (
	"github.com/xiajignge/aihub/config"
	"github.com/xiajignge/aihub/internal/dependencies"
	"github.com/xiajignge/aihub/internal/server"
	"github.com/xiajignge/aihub/internal/service"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		dependencies.Module,
		service.Module,
		server.Module,
	)
	app.Run()
}
