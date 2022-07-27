package cmd

import (
	"github.com/byzk-project-deploy/main-server/passwd"
	"github.com/byzk-project-deploy/main-server/sfnake"
	"github.com/byzk-project-deploy/main-server/shell"
	"github.com/byzk-project-deploy/main-server/ssh"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	transport_stream "github.com/go-base-lib/transport-stream"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"net"
	"os"
	"strings"
)

var (
	// systemDirPathVerifyHandle 系统目录路径验证
	systemDirPathVerifyHandle serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var targetPath *string
		if err := stream.ReceiveJsonMsg(&targetPath); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.New("缺失路径参数")
		}

		if stat, err := os.Stat(*targetPath); err == nil && stat.IsDir() {
			return nil, nil
		}

		return nil, serverclientcommon.ErrSystemPath.Newf("目录[%s]不存在", *targetPath)
	}

	// stemShellListHandle 当前系统可用的shell列表的获取
	stemShellListHandle serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		allowShellList := shell.AllowList()
		res := make([]string, len(allowShellList))
		for i := range allowShellList {
			res[i] = allowShellList[i] + " "
		}
		return serverclientcommon.NewExchangeDataByJson(allowShellList)
	}

	// systemShellCurrentGetHandle 系统调用获取当前系统的shell环境
	systemShellCurrentGetHandle serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		currentShell := strings.TrimSpace(viper.GetString("shell.current"))
		currentArgs := strings.TrimSpace(viper.GetString("shell.args"))

		return serverclientcommon.ExchangeData(currentShell + " " + currentArgs), nil
	}

	//systemShellCurrentSettingHandle 系统调用设置当前shell环境
	systemShellCurrentSettingHandle serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var options *serverclientcommon.ShellSettingOption
		if err := stream.ReceiveJsonMsg(&options); err != nil {
			return nil, serverclientcommon.ErrCodeValidation.New(err.Error())
		}

		if options.Name == "" {
			return nil, serverclientcommon.ErrCodeValidation.New("缺失的shell名称")
		}

		allowShellList := shell.AllowList()

		if !slices.Contains(allowShellList, options.Name) {
			return nil, serverclientcommon.ErrCodeValidation.New("未被允许设置的shell名称")
		}

		viper.Set("shell.current", options.Name)
		viper.Set("shell.args", strings.Join(options.Args, " "))
		return nil, viper.WriteConfig()
	}

	// systemCallHandle 系统命令调用处理
	systemCallHandle serverclientcommon.CmdHandler = func(stream *transport_stream.Stream, conn net.Conn) (serverclientcommon.ExchangeData, error) {
		var err error
		var callOption *serverclientcommon.SystemCallOption
		if err := stream.ReceiveJsonMsg(&callOption); err != nil {
			return nil, err
		}

		if callOption.Rand == "" {
			return nil, serverclientcommon.ErrCodeValidation.New("缺失交互参数")
		}

		if callOption.Name == "" {
			return nil, serverclientcommon.ErrCodeValidation.New("缺失的服务标识")
		}

		cliRand := callOption.Rand

		cId := sfnake.SFlake.GetIdStrUnwrap()
		passwd := passwd.Generator()
		passwd = cId + "_" + passwd

		callOption.Rand = passwd

		if callOption.Network, callOption.Addr, err = ssh.CurrentRemoteSSHAddr(callOption.Name); err != nil {
			return nil, err
		}

		ssh.AddPasswd(cId, passwd+cliRand)

		return serverclientcommon.NewExchangeDataByJson(callOption)

	}
)

//var (
//	// systemCallHandle 系统调用处理
//	systemCallHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, rw serverclientcommon.RWStreamInterface) *serverclientcommon.Result {
//
//		var callOption *serverclientcommon.SystemCallOption
//		if err := result.Data.Unmarshal(&callOption); err != nil {
//			return serverclientcommon.ErrDataParse.Result("数据包解析失败: " + err.Error())
//		}
//
//		if callOption.Rand == "" {
//			return serverclientcommon.ErrValidation.Result("缺失交互参数")
//		}
//
//		cliRand := callOption.Rand
//
//		cId := sfnake.SFlake.GetIdStrUnwrap()
//		passwd := passwd.Generator()
//		passwd = cId + "_" + passwd
//
//		callOption.Rand = passwd
//		callOption.Addr = ssh.ListenerPortStr()
//
//		ssh.AddPasswd(cId, passwd+cliRand)
//
//		return serverclientcommon.SuccessResult(callOption)
//	}
//)
