package ssh

import (
	"fmt"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/byzk-project-deploy/main-server/security"
	logs "github.com/byzk-worker/go-common-logs"
	"github.com/gliderlabs/ssh"
	"github.com/tjfoc/gmsm/gmtls"
	"net"
	"os"
	"path/filepath"
)

var unixFilePath = filepath.Join(os.TempDir(), ".bypt.ssh.socket")

var (
	listener net.Listener
)

var (
	listenerAddr string
)

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

	listenerAddr = ""

	if listener != nil {
		_ = listener.Close()
	}

	listener, err = net.Listen("tcp", config.Listener.Ip+":0")
	if err != nil {
		logs.Errorf("启动远程ssh监听失败: %s", err.Error())
		listener = nil
		return
	}

	listenerAddr = listener.Addr().String()

	go func() {
		logs.Infof("Remote Control SSH Server Run Success, Wait Accept...")
		if err = ssh.Serve(listener, sshCmdCallHandle, authOption); err != nil {
			logs.Errorf("远程监听启动失败: %s", err.Error())
		}
	}()
}

func init() {

	config.AddWatchAndNowExec(listenerServer)
	unixTlsConfig, err := security.Instance.GetTlsServerConfig("BYPT LOCAL SERVER", "unix", net.IPv4(127, 0, 0, 1))
	if err != nil {
		errors.ExitTlsError.Println("获取TLS配置失败: %s", err.Error())
	}

	_ = os.RemoveAll(unixFilePath)
	unixListener, err := net.Listen("unix", unixFilePath)
	if err != nil {
		errors.ExitSSHServerListener.Println("监听命令转发服务失败: " + err.Error())
	}
	unixListener = gmtls.NewListener(unixListener, unixTlsConfig)

	go func() {
		if err = ssh.Serve(unixListener, sshCmdCallHandle, authOption); err != nil {
			logs.Errorf("命令转发服务异常停止，请尝试重新启动: %s", err.Error())
			errors.ExitSSHServerListenerExit.Exit()
		}
		errors.ExitSSHServerListenerExit.Exit()
	}()
}

func CurrentRemoteSSHAddr(name string) (string, string, error) {
	if name == "unix" {
		return "unix", unixFilePath, nil
	}
	if listenerAddr == "" {
		return "", "", fmt.Errorf("未被允许远程请求")
	}
	return "tcp", listenerAddr, nil
}
