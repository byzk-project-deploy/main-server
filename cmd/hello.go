package cmd

import (
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
)

var helloCmdHandler serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, rw serverclientcommon.RWStreamInterface) *serverclientcommon.Result {
	var reqData string
	if err := result.Data.Unmarshal(&reqData); err != nil {
		panic(err)
	}
	return serverclientcommon.SuccessResult("hello " + reqData)
}
