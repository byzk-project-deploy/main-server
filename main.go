package main

import (
	"github.com/byzk-project-deploy/main-server/app"
	"github.com/byzk-project-deploy/main-server/config"
	"io"
	"os"
)

// import (
//
//	"github.com/byzk-project-deploy/main-server/app"
//	"github.com/byzk-project-deploy/main-server/config"
//	"io"
//	"os"
//
// )
//
// const userServiceTemplate = `[Unit]
// Description={{.Description}}
// ConditionFileIsExecutable={{.Path|cmdEscape}}
// {{range $i, $dep := .Dependencies}}
// {{$dep}} {{end}}
// [Service]
// StartLimitInterval=5
// StartLimitBurst=10
// ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
// {{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
// {{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
// {{if .UserName}}User={{.UserName}}{{end}}
// {{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
// {{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
// {{if and .LogOutput .HasOutputFileSupport -}}
// StandardOutput=file:/var/log/{{.Name}}.out
// StandardError=file:/var/log/{{.Name}}.err
// {{- end}}
// {{if gt .LimitNOFILE -1 }}LimitNOFILE={{.LimitNOFILE}}{{end}}
// {{if .Restart}}Restart={{.Restart}}{{end}}
// {{if .SuccessExitStatus}}SuccessExitStatus={{.SuccessExitStatus}}{{end}}
// RestartSec=120
// EnvironmentFile=-/etc/sysconfig/{{.Name}}
// [Install]
// WantedBy=default.target`
func main() {
	if len(os.Args) == 2 {
		if os.Args[1] == "--test" {
			io.WriteString(os.Stdout, config.BYPTServerVersion)
			return
		}
	}
	app.Run()
}

//func main() {
//	fmt.Println(123)
//}
