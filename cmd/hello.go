//go:build hello

package cmd

import (
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	transport_stream "github.com/go-base-lib/transport-stream"
	"net"
)

var helloCmdHandler serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
	var reqData string

	if err := stream.ReceiveJsonMsg(&reqData); err != nil {
		return nil, serverclientcommon.ErrCodeValidation.New("缺失请求参数")
	}

	return serverclientcommon.ExchangeData("World"), nil
}

func init() {
	serverclientcommon.CmdHello.Registry(helloCmdHandler)
}
