package plugins

import (
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	_ "github.com/byzk-project-deploy/main-server/db"
	"github.com/hashicorp/go-plugin"
	"os/exec"
)

func Manager() *manager {
	return m
}

func getPluginInfoByPath(p string) (*rpcinterfaces.PluginInfo, error) {
	pluginMap := map[string]plugin.Plugin{
		rpcinterfaces.PluginNameInfo: &rpcinterfaces.PluginInfoImpl{},
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		Plugins: pluginMap,
		Cmd:     exec.Command(p),
	})
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		panic(err)
	}

	raw, err := rpcClient.Dispense(rpcinterfaces.PluginNameInfo)
	if err != nil {
		return nil, err
	}

	applicationPlugin := raw.(rpcinterfaces.PluginInfoInterface)
	return applicationPlugin.Info()
}

func init() {
	m.init()
}
