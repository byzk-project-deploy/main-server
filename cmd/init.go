package cmd

import serverclientcommon "github.com/byzk-project-deploy/server-client-common"

func Init() {
	serverclientcommon.CmdSystemCall.Registry(systemCallHandle)
	serverclientcommon.CmdSystemShellList.Registry(stemShellListHandle)
	serverclientcommon.CmdSystemDirPath.Registry(systemDirPathVerifyHandle)
	serverclientcommon.CmdSystemShellCurrent.Registry(systemShellCurrentGetHandle)
	serverclientcommon.CmdSystemShellCurrentSetting.Registry(systemShellCurrentSettingHandle)
	serverclientcommon.CmdPluginInstall.Registry(pluginInstallCmd)
	serverclientcommon.CmdPluginList.Registry(pluginListCmd)
	serverclientcommon.CmdPluginInfoPromptList.Registry(pluginInfoPromptCmd)
	serverclientcommon.CmdPluginInfo.Registry(pluginInfoCmd)
	serverclientcommon.CmdRemoteServerAdd.Registry(remoteServerJoin)
	serverclientcommon.CmdRemoteServerList.Registry(remoteServerList)
	serverclientcommon.CmdRemoteServerInfo.Registry(remoteServerInfo)
	serverclientcommon.CmdRemoteServerUpdate.Registry(remoteServerUpdate)
	serverclientcommon.CmdRemoteServerUpdateAlias.Registry(remoteServerAliasUpdate)
	serverclientcommon.CmdRemoteServerDel.Registry(remoteServerRemove)
	serverclientcommon.CmdRemoteServerRepair.Registry(remoteServerRepair)
	serverclientcommon.CmdRemoteServerFileUpload.Registry(remoteServerUpload)
	serverclientcommon.CmdRemoteServerFileDownload.Registry(remoteServerDownload)
}
