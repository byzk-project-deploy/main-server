package server

import (
	"bytes"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	"text/template"
)

const _remoteConfigTemplate = `[database]
path = '{{.homedir}}/.bypt/db/local.data'

[listener]
allowremotecontrol = true
ip = '{{.ip}}'
port = '65526'

[logs]
level = 'info'
path = '{{.homedir}}/.bypt/logs'

[plugin]
storepath = '{{.homedir}}/.bypt/plugins'

[shell]
allowshelllist = []
allowshelllistfile = '/etc/shells'
args = '-i -c'
current = '/usr/bin/bash'`

var (
	remoteConfigTml *template.Template

	remoteInstallPathTml *template.Template
)

func execTemplateToStr(tml *template.Template, data any) (string, error) {
	buf, err := execTemplateToBuf(tml, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func execTemplateToBuf(tml *template.Template, data any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	return buf, tml.Execute(buf, data)
}

func init() {
	var err error

	config.AddWatchAndNowExec(func(config *config.Info) {
		rawRemoteInstallPathTml := remoteInstallPathTml
		if remoteInstallPathTml, err = template.New("config_path_template").Parse(config.Remote.InstallPath); err != nil && rawRemoteInstallPathTml != nil {
			remoteInstallPathTml = rawRemoteInstallPathTml
		}

	})

	remoteConfigTml, err = template.New("config_template").Parse(_remoteConfigTemplate)
	if err != nil {
		errors.ExitConfigFileParser.Println("解析远程模板失败: %s", err.Error())
	}
}
