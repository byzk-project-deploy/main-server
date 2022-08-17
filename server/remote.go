package server

import (
	"bufio"
	"fmt"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/security"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	transportstream "github.com/go-base-lib/transport-stream"
	"github.com/tjfoc/gmsm/gmtls"
	"io"
	"net"
	"time"
)

type remoteServerConn struct {
	s     *serverclientcommon.ServerInfo
	conn  net.Conn
	err   error
	bufRW *bufio.ReadWriter
}

func (r *remoteServerConn) ConnToStream() (*transportstream.Stream, error) {
Start:
	if r.err != nil {
		return nil, r.err
	}

	if r.conn == nil {
		r.err = r.Connection()
		goto Start
	}

	stream := transportstream.NewStream(r.bufRW)

	if output, err := serverclientcommon.CmdHello.Exchange(stream); err != nil {
		if err == io.EOF {
			r.Close()
			goto Start
		}
		r.err = err
		return nil, err
	} else if string(output) != config.BYPTServerVersion {
		r.Close()
		r.err = fmt.Errorf("非法的服务器信息, 请尝试重新安装")
		r.s.Status = serverclientcommon.ServerStatusNetworkErr
		r.s.EndMsg = r.err.Error()
		Manager.updateServerInfoToStore(r.s)
		return nil, r.err
	}

	return stream, nil
}

func (r *remoteServerConn) Ping() error {
	_, err := r.ConnToStream()
	return err
}

func (r *remoteServerConn) UpdateServerInfo(s *serverclientcommon.ServerInfo) {
	r.Close()
	r.s = s
}

func (r *remoteServerConn) Reconnection() error {
	r.Close()
	return r.Connection()
}

// Connection 连接远端服务，连接失败返回错误
func (r *remoteServerConn) Connection() (err error) {
	defer func() {
		r.err = err
	}()

	serverInfo := r.s
	if serverInfo.Status < serverclientcommon.ServerStatusNetworkErr {
		return fmt.Errorf("当前服务的状态无法进行远程连接")
	}

	if serverInfo.IP == nil {
		return fmt.Errorf("服务器IP地址不能为空")
	}

	if serverInfo.Port == 0 {
		return fmt.Errorf("缺失端口信息")
	}

	r.Close()

	tlsConfig, err := security.Instance.GetTlsClientConfig(security.GetRemoteServerName(serverInfo.IP), security.LinkRemoteDnsFlag, serverInfo.IP)
	if err != nil {
		return fmt.Errorf("创建连接凭据失败: %s", err.Error())
	}

	if r.conn, err = gmtls.DialWithDialer(&net.Dialer{
		Timeout: time.Second * 5,
	}, "tcp", fmt.Sprintf("%s:%d", serverInfo.IP.String(), serverInfo.Port), tlsConfig); err != nil {
		r.conn = nil
		return fmt.Errorf("%s", err.Error())
	}
	r.bufRW = bufio.NewReadWriter(bufio.NewReader(r.conn), bufio.NewWriter(r.conn))

	return nil
}

// Close 关闭连接
func (r *remoteServerConn) Close() {
	r.err = nil
	r.bufRW = nil
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}

func newRemoteServerConn(s *serverclientcommon.ServerInfo) *remoteServerConn {
	return &remoteServerConn{
		s: s,
	}
}
