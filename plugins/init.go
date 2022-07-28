package plugins

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	"github.com/go-base-lib/coderutils"
	"github.com/hashicorp/go-plugin"
	"os"
	"os/exec"
)

func installPlugin(pluginPath, targetPath string) error {
	stat, err := os.Stat(pluginPath)
	if err != nil || stat.IsDir() {
		return fmt.Errorf("插件路径[%s]不存在", pluginPath)
	}

	pluginInfo, err := getPluginInfoByPath(pluginPath)
	if err != nil {
		return fmt.Errorf("获取插件[%s]的信息失败: %s", pluginPath, err.Error())
	}

	file, err := os.OpenFile(pluginPath, os.O_RDONLY, 0655)
	if err != nil {
		return fmt.Errorf("打开插件文件[%s]失败: %s", pluginPath, err.Error())
	}
	defer file.Close()

	md5Res, err := coderutils.HashByReader(md5.New(), file)
	if err != nil {
		return fmt.Errorf("获取插件[%s]的MD5摘要失败: %s", pluginPath, err.Error())
	}

	if _, err = file.Seek(0, 0); err != nil {
		return err
	}

	sha1Res, err := coderutils.HashByReader(sha1.New(), file)
	if err != nil {
		return fmt.Errorf("获取插件[%s]的SHA1摘要失败: %s", pluginPath, err.Error())
	}

	fmt.Println(md5Res.ToHexStr())
	fmt.Println(sha1Res.ToHexStr())
	fmt.Println(pluginInfo)

	//file, err := os.OpenFile(pluginPath, os.O_RDONLY, 0655)
	//if err != nil {
	//	return fmt.Errorf("打开插件文件[%s]失败: %s", pluginPath, err.Error())
	//}
	//defer file.Close()
	//
	return nil
}

func getPluginInfoByPath(p string) (*rpcinterfaces.PluginInfo, error) {
	pluginMap := map[string]plugin.Plugin{
		rpcinterfaces.PluginNameInfo: &rpcinterfaces.PluginInfoImpl{},
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "BASIC_PLUGIN",
			MagicCookieValue: "hello",
		},

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

func InitPlugins() {
	//coder.Sm4RandomKey()
}

//func init() {
//	logger := hclog.New(&hclog.LoggerOptions{
//		Name:   "plugin",
//		Output: os.Stdout,
//		Level:  hclog.Debug,
//	})
//
//	pluginMap := map[string]plugin.Plugin{
//		rpcinterfaces.PluginNameInfo: &rpcinterfaces.PluginInfoImpl{},
//	}
//
//	client := plugin.NewClient(&plugin.ClientConfig{
//		HandshakeConfig: plugin.HandshakeConfig{
//			ProtocolVersion:  1,
//			MagicCookieKey:   "BASIC_PLUGIN",
//			MagicCookieValue: "hello",
//		},
//
//		Plugins: pluginMap,
//		Cmd:     exec.Command("/home/slx/works/03-byzk-project-deploy/plugin-docker/test"),
//		Logger:  logger,
//	})
//	defer client.Kill()
//
//	rpcClient, err := client.Client()
//	if err != nil {
//		panic(err)
//	}
//
//	raw, err := rpcClient.Dispense(rpcinterfaces.PluginNameInfo)
//	if err != nil {
//		panic(err)
//	}
//
//	applicationPlugin := raw.(rpcinterfaces.PluginInfoInterface)
//	info, err := applicationPlugin.Info()
//	if err == rpcinterfaces.ErrTimeout {
//		return
//	}
//	fmt.Println(info)
//
//}
