package plugins

import (
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	"github.com/byzk-project-deploy/go-plugin"
	_ "github.com/byzk-project-deploy/main-server/db"
	"os/exec"
)

func Manager() *manager {
	return m
}

func getPluginInfoByPath(p string) (*rpcinterfaces.PluginInfo, error) {
	pluginMap := map[string]plugin.Plugin{
		rpcinterfaces.PluginNameBase: &rpcinterfaces.PluginBaseImpl{},
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

	raw, err := rpcClient.Dispense(rpcinterfaces.PluginNameBase)
	if err != nil {
		return nil, err
	}

	applicationPlugin := raw.(rpcinterfaces.PluginBaseInterface)
	return applicationPlugin.Info()
}

func init() {
	m.init()
}
