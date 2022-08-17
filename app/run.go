package app

import (
	"github.com/byzk-project-deploy/main-server/cmd"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/db"
	"github.com/byzk-project-deploy/main-server/logs"
	"github.com/byzk-project-deploy/main-server/plugins"
	"github.com/byzk-project-deploy/main-server/server"
)

func Run() {
	config.Init()
	db.Init()
	cmd.Init()
	logs.Init()
	plugins.Init()

	server.Run()
}
