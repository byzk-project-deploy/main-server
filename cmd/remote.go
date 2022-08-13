package cmd

import (
	"github.com/byzk-project-deploy/main-server/server"
	"github.com/byzk-project-deploy/main-server/vos"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/byzk-worker/go-db-utils/sqlite"
	transportstream "github.com/go-base-lib/transport-stream"
	"net"
)

var (
	// remoteServerJoin 远程服务添加
	remoteServerJoin serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {

		var remoteIpList []string
		if err := stream.ReceiveJsonMsg(&remoteIpList); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取要添加的服务器IP失败")
		}

		remoteServerIpLen := len(remoteIpList)
		if remoteServerIpLen == 0 {
			return serverclientcommon.NewExchangeDataByJson(remoteIpList)
		}

		dbServerInfoModel := sqlite.Db().Model(&vos.DbServerInfo{})
		count := 0
		for i := 0; i < remoteServerIpLen; i++ {
			ipStr := remoteIpList[i]
			if err := dbServerInfoModel.Where("id=?", ipStr).Count(&count).Error; err != nil {
				serverclientcommon.ErrServerInside.Newf("查询服务器数据失败: %s", err.Error()).WriteTo(stream)
				continue
			}

			if count != 0 {
				continue
			}

			ip := net.ParseIP(ipStr)
			if ip == nil {
				serverclientcommon.ErrCodeValidation.Newf("不符合规范的ip格式: %s", ipStr).WriteTo(stream)
				continue
			}

			if err := stream.WriteMsg([]byte(ipStr), transportstream.MsgFlagSuccess); err != nil {
				serverclientcommon.ErrorByErr(err).WriteTo(stream)
				continue
			}

			sshPortBytes, err := stream.ReceiveMsg()
			if err != nil {
				serverclientcommon.ErrorByErr(err).WriteTo(stream)
				continue
			}

			sshPort, err := transportstream.BytesToInt[uint16](sshPortBytes)
			if err != nil {
				serverclientcommon.ErrCodeValidation.Newf("获取服务器的SSH端口失败: %s", err.Error()).WriteTo(stream)
				continue
			}

			usernameBytes, err := stream.ReceiveMsg()
			if err != nil {
				serverclientcommon.ErrorByErr(err).WriteTo(stream)
				continue
			}

			userPasswordBytes, err := stream.ReceiveMsg()
			if err != nil {
				serverclientcommon.ErrorByErr(err).WriteTo(stream)
				continue
			}

			username := string(usernameBytes)
			userPassword := string(userPasswordBytes)

			if _, err = server.Manager.AddServer(&serverclientcommon.ServerInfo{
				Id:          ipStr,
				IP:          ip,
				SSHUser:     username,
				SSHPassword: userPassword,
				SSHPort:     sshPort,
			}); err != nil {
				serverclientcommon.ErrorByErr(err).WriteTo(stream)
				continue
			}

			_ = stream.WriteMsg(nil, transportstream.MsgFlagSuccess)
		}

		return nil, nil
	}

	// remoteServerList 远程服务器列表查询
	remoteServerList serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var searchKeywords []string
		_ = stream.ReceiveJsonMsg(&searchKeywords)

		return serverclientcommon.NewExchangeDataByJson(server.Manager.List(searchKeywords))
	}

	// remoteServerInfo 远程服务器信息
	remoteServerInfo serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var serverName string
		if err := stream.ReceiveJsonMsg(&serverName); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取服务器名称失败: %s", err.Error())
		}

		serverInfo, err := server.Manager.FindByServerName(serverName)
		if err != nil {
			return nil, err
		}

		return serverclientcommon.NewExchangeDataByJson(serverInfo)
	}

	// remoteServerUpdate 远端服务器信息更新
	remoteServerUpdate serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var serverInfo *serverclientcommon.ServerInfo
		if err := stream.ReceiveJsonMsg(&serverInfo); err != nil {
			return nil, err
		}
		return nil, server.Manager.UpdateServerInfo(serverInfo)
	}

	// remoteServerAliasUpdate 服务器别名设置与更新
	remoteServerAliasUpdate serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var (
			aliasList []string
			ip        string
		)

		if ipBytes, err := stream.ReceiveMsg(); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取服务器IP失败: %s", err.Error())
		} else {
			ip = string(ipBytes)
		}

		if err := stream.ReceiveJsonMsg(&aliasList); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取别名列表失败: %s", err.Error())
		}

		return nil, server.Manager.SettingAlias(aliasList, ip)
	}

	// remoteServerRemove 服务器移除
	remoteServerRemove serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var serverName string
		if err := stream.ReceiveJsonMsg(&serverName); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取要移除的服务器标识失败: %s", err.Error())
		}

		return nil, server.Manager.RemoveServer(serverName)
	}
)
