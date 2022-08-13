package main

import (
	_ "github.com/byzk-project-deploy/main-server/cmd"
	"github.com/byzk-project-deploy/main-server/config"
	_ "github.com/byzk-project-deploy/main-server/db"
	_ "github.com/byzk-project-deploy/main-server/logs"
	"github.com/byzk-project-deploy/main-server/server"
	"io"
	"os"
)

func main() {
	if len(os.Args) == 2 {
		if os.Args[1] == "--test" {
			io.WriteString(os.Stdout, config.BYPTServerVersion)
			return
		}
	}
	server.Run()
}
