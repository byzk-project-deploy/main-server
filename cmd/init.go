package cmd

import serverclientcommon "github.com/byzk-project-deploy/server-client-common"

func init() {
	serverclientcommon.CmdHello.Registry(helloCmdHandler)
	serverclientcommon.CmdSystemCall.Registry(systemCallHandle)
	serverclientcommon.CmdSystemShellList.Registry(stemShellListHandle)
	serverclientcommon.CmdSystemDirPath.Registry(systemDirPathVerifyHandle)
	serverclientcommon.CmdSystemShellCurrent.Registry(systemShellCurrentGetHandle)
	serverclientcommon.CmdSystemShellCurrentSetting.Registry(systemShellCurrentSettingHandle)
}
