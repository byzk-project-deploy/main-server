package cmd

import (
	"fmt"
	"github.com/byzk-project-deploy/main-server/server"
	"github.com/byzk-project-deploy/main-server/ssh"
	"github.com/byzk-project-deploy/main-server/vos"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/byzk-worker/go-db-utils/sqlite"
	transportstream "github.com/go-base-lib/transport-stream"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
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

	// remoteServerRepair 远程服务自动修复
	remoteServerRepair serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		return serverclientcommon.NewExchangeDataByJson(server.Manager.Repair())
	}

	// remoteServerUpload 文件上传
	remoteServerUpload serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var uploadReqData *serverclientcommon.RemoteServerUploadRequest
		if err := stream.ReceiveJsonMsg(&uploadReqData); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取上传信息失败: %s", err.Error())
		}

		sourceAddr := uploadReqData.SourceAddr
		if sourceAddr.Path == "" {
			return nil, serverclientcommon.ErrCodeValidation.New("源文件地址不能为空")
		}

		var (
			stat os.FileInfo
			err  error
		)
		if sourceAddr.Server != "" {
			if stat, err = server.Manager.ReadFileInfo(sourceAddr.Server, sourceAddr.Path, uploadReqData.UploadType); err != nil {
				return nil, err
			}
		} else {
			if stat, err = os.Stat(sourceAddr.Path); err != nil {
				return nil, serverclientcommon.ErrCodeValidation.Newf("打开本地路径[%s]失败: %s", sourceAddr.Path, err.Error())
			}
		}

		if stat.IsDir() && !uploadReqData.Recursive {
			return nil, fmt.Errorf("不允许上传目录")
		}

		if err = stream.WriteMsg(nil, transportstream.MsgFlagSuccess); err != nil {
			return nil, err
		}

		if _, err = stream.ReceiveMsg(); err != nil {
			return nil, err
		}

		uploadFn, err := remoteUploadToServer(uploadReqData.TargetAddrList, stream)
		if err != nil {
			return nil, err
		}

		if sourceAddr.Server != "" {
			err = server.Manager.RangeFile(sourceAddr.Server, sourceAddr.Path, uploadReqData.UploadType, uploadFn)
		} else {
			err = remoteUploadRangeLocalFile(sourceAddr.Path, "", uploadFn)
		}

		return nil, err
	}

	// remoteServerDownload 远程服务文件下载
	remoteServerDownload serverclientcommon.CmdHandler = func(stream *transportstream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		tBytes, err := stream.ReceiveMsg()
		if err != nil {
			return nil, err
		}

		_t, err := transportstream.BytesToInt[uint8](tBytes)
		t := serverclientcommon.UploadType(_t)

		if t > serverclientcommon.UploadUnknown || t < 0 {
			return nil, fmt.Errorf("不支持的文件传输方式")
		}

		remoteAddrStr, err := stream.ReceiveMsg()
		if err != nil {
			return nil, err
		}

		remoteAddrFlags := strings.Split(string(remoteAddrStr), "@")
		if len(remoteAddrFlags) != 2 {
			return nil, fmt.Errorf("地址[%s]格式不正确", remoteAddrStr)
		}

		return nil, server.Manager.RangeFile(remoteAddrFlags[0], remoteAddrFlags[1], t, func(filename string, relativePath string, filesize int64, reader io.Reader) error {
			if filesizeBytes, err := transportstream.IntToBytes(filesize); err != nil {
				return err
			} else {
				if err := stream.WriteMsgStream([]byte(filename), transportstream.MsgFlagSuccess).
					WriteMsgStream([]byte(relativePath), transportstream.MsgFlagSuccess).
					WriteMsgStream(filesizeBytes, transportstream.MsgFlagSuccess).Error(); err != nil {
					return err
				}
			}

			if _, err := stream.ReceiveMsg(); err != nil {
				return err
			}

			buf := make([]byte, 1024*1024*5)
			for {
				n, err := reader.Read(buf)
				if err == io.EOF {
					if n > 0 {
						goto Write
					}
					return nil
				}

				if err != nil {
					return err
				}

			Write:
				if err = stream.WriteMsg(buf[:n], transportstream.MsgFlagSuccess); err != nil {
					return err
				}

			}
		})
	}
)

func remoteUploadRangeLocalFile(path, relativePath string, fn server.RangeFileFn) error {
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("获取本地路径[%s]的文件信息失败: %s", path, err.Error())
	}

	if stat.IsDir() {
		dirChild, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("获取本地目录[%s]中的子文件信息失败: %s", path, err.Error())
		}
		for i := range dirChild {
			child := dirChild[i]
			if err = remoteUploadRangeLocalFile(filepath.Join(path, child.Name()), filepath.Join(relativePath, child.Name()), fn); err != nil {
				return err
			}
		}
		return nil
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0655)
	if err != nil {
		return fmt.Errorf("打开文件[%s]失败: %s", path, err.Error())
	}
	defer f.Close()
	return fn(f.Name(), relativePath, stat.Size(), f)
}

func remoteUploadToServer(serverAddrList []*serverclientcommon.UploadAddrInfo, stream *transportstream.Stream) (server.RangeFileFn, error) {
	serverFtpList := make([]*ssh.SSHSftpWrapper, 0, len(serverAddrList))
	for i := range serverAddrList {
		addr := serverAddrList[i]
		serverInfo, err := server.Manager.FindByServerName(addr.Server)
		if err != nil {
			return nil, err
		}
		ftpWrapper, err := server.Manager.GetServerFtp(serverInfo)
		if err != nil {
			return nil, err
		}
		addr.Path = ftpWrapper.PathReplace(addr.Path)
		serverFtpList = append(serverFtpList, ftpWrapper)
	}
	return func(filename, relativePath string, filesize int64, reader io.Reader) error {
		if err := stream.WriteMsg([]byte(filename), transportstream.MsgFlagSuccess); err != nil {
			return err
		}

		if _, err := stream.ReceiveMsg(); err != nil {
			return err
		}

		writerList := make([]io.WriteCloser, 0, len(serverFtpList))
		defer func() {
			for i := range writerList {
				writerList[i].Close()
			}
		}()
		for i := range serverFtpList {
			sftpWrapper := serverFtpList[i]
			if relativePath != "" {
				_ = sftpWrapper.MkdirAll(filepath.Join(serverAddrList[i].Path, filepath.Dir(relativePath)))
			} else {
				_ = sftpWrapper.MkdirAll(filepath.Dir(serverAddrList[i].Path))
			}
			f, err := sftpWrapper.OpenFile(filepath.Join(serverAddrList[i].Path, relativePath), os.O_WRONLY|os.O_CREATE)
			if err != nil {
				return fmt.Errorf("服务器[%s]创建文件[%s]失败, 上传终止, 错误原因: %s", serverAddrList[i].Server, filepath.Join(serverAddrList[i].Path, relativePath), err.Error())
			}
			writerList = append(writerList, f)
		}

		okCount := 0.0
		buf := make([]byte, 1024*1024*5)
		for {
			n, err := reader.Read(buf)
			if err == io.EOF {
				if n > 0 {
					goto Write
				}
				stream.WriteJsonMsg(&serverclientcommon.RemoteServerUploadResponse{
					Success:  true,
					Progress: 100,
					End:      true,
				})
				return nil
			}

			if err != nil {
				return fmt.Errorf("读取文件[%s]内容失败: %s", filename, err.Error())
			}

		Write:
			for i := len(writerList) - 1; i >= 0; i-- {
				writer := writerList[i]
				if _, err = writer.Write(buf[:n]); err != nil {
					writer.Close()
					writerList = append(writerList[:i], writerList[i+1:]...)
					serverAddrInfo := serverAddrList[i]
					serverAddrList = append(serverAddrList[:i], serverAddrList[i+1:]...)
					if err = stream.WriteJsonMsg(&serverclientcommon.RemoteServerUploadResponse{
						Success:  false,
						ServerIp: serverAddrInfo.Server,
						ErrMsg:   fmt.Sprintf("向服务器[%s]上传文件[%s]失败, 错误原因: %s", serverAddrInfo.Server, filename, err.Error()),
					}); err != nil {
						return err
					}
				}
			}

			okCount += float64(n)

			if err = stream.WriteJsonMsg(&serverclientcommon.RemoteServerUploadResponse{
				Success:  true,
				Progress: int(math.Floor(okCount / float64(filesize) * 100)),
			}); err != nil {
				return err
			}

		}
	}, nil
}
