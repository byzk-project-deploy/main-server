package server

import (
	"bufio"
	"github.com/byzk-project-deploy/main-server/security"
	transport_stream "github.com/go-base-lib/transport-stream"
	"github.com/tjfoc/gmsm/gmtls"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	logs "github.com/byzk-worker/go-common-logs"
)

var (
	listener     net.Listener
	unixListener net.Listener
)

var (
	listenerAddr string
)

var unixFilePath = filepath.Join(os.TempDir(), ".bypt.socket")

func init() {
	_ = os.RemoveAll(unixFilePath)
}

func listenerServer(config *config.Info) {
	var err error

	if !config.Listener.AllowRemoteControl {
		if listener != nil {
			_ = listener.Close()
			logs.Infof("远程控制服务已成功停止, 地址: %s", listenerAddr)
		}
		listenerAddr = ""
		return
	}

	if listener != nil {
		_ = listener.Close()
	}

	tlsConfig, err := security.Instance.GetTlsServerConfig(security.GetRemoteServerName(net.ParseIP(config.Listener.Ip)), security.LinkRemoteDnsFlag, net.ParseIP(config.Listener.Ip))
	if err != nil {
		logs.Errorf("启动远程监听失败, 创建认证凭证失败: %s", err.Error())
	}

	tempListenerAddr := config.Listener.Ip + ":" + strconv.FormatUint(config.Listener.Port, 10)
	if tempListenerAddr == listenerAddr {
		return
	}
	if listener != nil {
		_ = listener.Close()
	}
	listenerAddr = tempListenerAddr

	listener, err = net.Listen("tcp", listenerAddr)
	if err != nil {
		logs.Errorf("启动远程监听失败: %s", err)
		return
	}

	listener = gmtls.NewListener(listener, tlsConfig)

	logs.Infof("Remote Control Server Run Success, Server Listener Address: %s", listenerAddr)
	go listenerHandle("远程控制", false, listener)
}

// listenerHandle 监听处理器
func listenerHandle(serverName string, endExit bool, listener net.Listener) {
	for {
		accept, err := listener.Accept()
		if err != nil {
			if err == io.EOF {
				if endExit {
					logs.Infof("服务[%s], 监听地址: [%s], 正常退出", serverName, listener.Addr().String())
					errors.ExitServerListenerExit.Exit()
				}
				return
			}

			if endExit {
				logs.Errorf("服务[%s]异常退出", err.Error())
				errors.ExitServerListenerExit.Exit()
			}

			return
		}
		go connHandle(accept)
	}
}

func connHandle(conn net.Conn) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	stream := transport_stream.NewStream(rw)
	for {
		if err := serverclientcommon.CmdRoute(stream, conn); err == io.EOF {
			return
		}
	}
}

func Run() {
	var err error

	config.AddWatchAndNowExec(listenerServer)
	unixTlsConfig, err := security.Instance.GetTlsServerConfig(security.LinkLocalFlag, "unix", net.IPv4(127, 0, 0, 1))
	if err != nil {
		errors.ExitTlsError.Println("获取TLS配置失败: %s", err.Error())
	}
	unixListener, err = net.Listen("unix", unixFilePath)
	if err != nil {
		errors.ExitUnixSocketListener.Println("监听本地通信交互文件失败: %s", err.Error())
	}
	unixListener = gmtls.NewListener(unixListener, unixTlsConfig)
	logs.Info("^_^ Local Server Run Success ^-^")
	listenerHandle("本地服务", true, unixListener)
}
