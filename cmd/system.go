package cmd

import (
	"os"
	"strings"

	"github.com/byzk-project-deploy/main-server/passwd"
	"github.com/byzk-project-deploy/main-server/sfnake"
	"github.com/byzk-project-deploy/main-server/shell"
	"github.com/byzk-project-deploy/main-server/ssh"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/spf13/viper"
)

var (
	// systemDirPathVerifyHandle 验证系统目录是否存在
	systemDirPathVerifyHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, conn serverclientcommon.RWStreamInterface) *serverclientcommon.Result {
		var targetPath *string
		if err := result.Data.Unmarshal(&targetPath); err != nil {
			return serverclientcommon.ErrValidation.Result("缺失路径参数")
		}

		exist := false
		if stat, err := os.Stat(*targetPath); err == nil && stat.IsDir() {
			exist = true
		}
		return serverclientcommon.SuccessResult(exist)
	}
	// systemCallHandle 系统调用处理
	systemCallHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, rw serverclientcommon.RWStreamInterface) *serverclientcommon.Result {

		var callOption *serverclientcommon.SystemCallOption
		if err := result.Data.Unmarshal(&callOption); err != nil {
			return serverclientcommon.ErrDataParse.Result("数据包解析失败: " + err.Error())
		}

		if callOption.Rand == "" {
			return serverclientcommon.ErrValidation.Result("缺失交互参数")
		}

		cliRand := callOption.Rand

		cId := sfnake.SFlake.GetIdStrUnwrap()
		passwd := passwd.Generator()
		passwd = cId + "_" + passwd

		callOption.Rand = passwd
		callOption.Addr = ssh.ListenerPortStr()

		ssh.AddPasswd(cId, passwd+cliRand)

		return serverclientcommon.SuccessResult(callOption)
	}

	// stemShellListHandle 系统可用shell列表获取
	stemShellListHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, conn serverclientcommon.RWStreamInterface) *serverclientcommon.Result {
		allowShellList := shell.AllowList()
		res := make([]string, len(allowShellList))
		for i := range allowShellList {
			res[i] = allowShellList[i] + " "
		}
		return serverclientcommon.SuccessResult(res)
	}

	// systemShellCurrentGetHandle 系统调用当前shell环境获取
	systemShellCurrentGetHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, conn serverclientcommon.RWStreamInterface) *serverclientcommon.Result {
		currentShell := strings.TrimSpace(viper.GetString("shell.current"))
		currentShellArgs := strings.TrimSpace(viper.GetString("shell.args"))

		return serverclientcommon.SuccessResult(currentShell + " " + currentShellArgs)
	}

	// systemShellCurrentSettingHandle 系统调用设置当前shell环境
	systemShellCurrentSettingHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, conn serverclientcommon.RWStreamInterface) *serverclientcommon.Result {
		var options *serverclientcommon.ShellSettingOption
		if err := result.Data.Unmarshal(&options); err != nil {
			return serverclientcommon.ErrValidation.Result(err.Error())
		}

		if options.Name == "" {
			return serverclientcommon.ErrValidation.Result("缺失shell名称")
		}

		allowShellList := shell.AllowList()

		for i := range allowShellList {
			if allowShellList[i] == options.Name {
				goto End
			}
		}
		return serverclientcommon.ErrValidation.Result("未被允许的shell名称")
	End:
		viper.Set("shell.current", options.Name)
		viper.Set("shell.args", strings.Join(options.Args, " "))
		viper.WriteConfig()

		return serverclientcommon.SuccessResultEmpty()
	}
)
