package cmd

import (
	"github.com/byzk-project-deploy/main-server/config"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	transport_stream "github.com/go-base-lib/transport-stream"
	"net"
)

var helloCmdHandler serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
	return serverclientcommon.ExchangeData(config.BYPTServerVersion), nil
}

func init() {
	serverclientcommon.CmdHello.Registry(helloCmdHandler)
}
