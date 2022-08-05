//go:build inside

package cmd

import (
	"github.com/byzk-project-deploy/main-server/server"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	transport_stream "github.com/go-base-lib/transport-stream"
	"net"
	"strings"
)

var (
	// keypairCmd 密钥创建命令
	keypairCmd serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var options *serverclientcommon.KeypairGeneratorInfo
		if err := stream.ReceiveJsonMsg(&options); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.Newf("获取参数失败: %s", err.Error())
		}

		var (
			cert *server.CertResult
			err  error
		)

		t := strings.ToLower(options.Type)
		switch t {
		case "plugin":
			if options.Author == "" {
				return nil, serverclientcommon.ErrCodeValidation.New("作者名称不能为空")
			}

			if options.Name == "" {
				return nil, serverclientcommon.ErrCodeValidation.New("插件名称不能为空")
			}

			cert, err = server.TlsManager.GeneratorClientPemCert(options.Author, options.Name+".bypt", net.IPv4(127, 0, 0, 1))

		case "client":
			cert, err = server.TlsManager.GeneratorClientPemCert("BYPT LOCAL SERVER", "unix", net.IPv4(127, 0, 0, 1))
		default:
			return nil, serverclientcommon.ErrCodeValidation.Newf("非法的密钥类型: %s", t)
		}

		if err != nil {
			return nil, serverclientcommon.ErrServerInside.Newf("密钥生成失败: %s", err.Error())
		}

		return serverclientcommon.NewExchangeDataByJson(cert)
	}
)

func init() {
	serverclientcommon.CmdKeyPair.Registry(keypairCmd)
}
