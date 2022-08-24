package cmd

import (
	"github.com/byzk-project-deploy/main-server/plugins"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	transportstream "github.com/go-base-lib/transport-stream"
	"net"
)

var (
	// pluginInstallCmd 插件安装命令
	pluginInstallCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var pluginPath string
		if err := stream.ReceiveJsonMsg(&pluginPath); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取插件路径失败: %s", err.Error())
		}

		info, err := plugins.Manager().Install(pluginPath)
		if err != nil {
			return nil, err
		}
		return serverclientcommon.NewExchangeDataByJson(info)
	}

	// pluginUninstallCmd 插件卸载
	pluginUninstallCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var idOrName string
		if err := stream.ReceiveJsonMsg(&idOrName); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取插件的标识失败: %s", err.Error())
		}

		return nil, plugins.Manager().Uninstall(idOrName)
	}

	// pluginEnabledCmd 插件启动
	pluginEnabledCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var idOrName string
		if err := stream.ReceiveJsonMsg(&idOrName); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取插件标识失败: %s", err.Error())
		}

		return nil, plugins.Manager().Start(idOrName)
	}

	// pluginDisabledCmd 插件禁用命令
	pluginDisabledCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var idOrName string
		if err := stream.ReceiveJsonMsg(&idOrName); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取插件标识失败: %s", err.Error())
		}

		return nil, plugins.Manager().Stop(idOrName)
	}

	// pluginListCmd 插件列表查询命令
	pluginListCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		return serverclientcommon.NewExchangeDataByJson(plugins.Manager().StatusInfoList())
	}

	// pluginInfoPromptCmd 插件信息检索提示列表获取
	pluginInfoPromptCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var prefix string
		if err := stream.ReceiveJsonMsg(&prefix); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("接受插件索引前缀失败: %s", err.Error())
		}
		return serverclientcommon.NewExchangeDataByJson(plugins.Manager().FindStatusInfoByIdOrName(prefix))
	}

	// pluginInfoCmd 插件详细信息查询命令
	pluginInfoCmd serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var idOrName string
		if err := stream.ReceiveJsonMsg(&idOrName); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("接收插件ID失败: %s", err.Error())
		}

		info, err := plugins.Manager().FindOneStatusInfoByIdOrName(idOrName)
		if err != nil {
			return nil, err
		}
		return serverclientcommon.NewExchangeDataByJson(info)
	}
)
