package main

import (
	_ "github.com/byzk-project-deploy/main-server/cmd"
	_ "github.com/byzk-project-deploy/main-server/db"
	_ "github.com/byzk-project-deploy/main-server/logs"
	"github.com/byzk-project-deploy/main-server/server"
)

func main() {
	server.Run()
}
