package cmd

import serverclientcommon "github.com/byzk-project-deploy/server-client-common"

func init() {
	serverclientcommon.CmdHello.Registry(helloCmdHandler)
	serverclientcommon.CmdSystemCall.Registry(systemCallHandle)
}
