package cmd

import (
	"github.com/byzk-project-deploy/main-server/passwd"
	"github.com/byzk-project-deploy/main-server/sfnake"
	"github.com/byzk-project-deploy/main-server/ssh"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
)

// func init() {
// 	serverclientcommon.CmdSystemCall.Registry(systemCallHandle)
// }

// type serverCmdStdinWrapper struct {
// 	w        io.WriteCloser
// 	rw       serverclientcommon.RWStreamInterface
// 	lineBuf  *bytes.Buffer
// 	stopFlag chan struct{}
// }

// func (s *serverCmdStdinWrapper) Start() {
// 	s.lineBuf = &bytes.Buffer{}
// 	s.stopFlag = make(chan struct{}, 1)
// 	for {
// 		l, isPrefix, err := s.rw.ReadLine()
// 		if err != nil {
// 			s.stopFlag <- struct{}{}
// 			return
// 		}
// 		if isPrefix {
// 			s.lineBuf.Write(l)
// 			continue
// 		}

// 		r := &serverclientcommon.Result{}
// 		if err = r.Parse(l); err != nil {
// 			logs.Errorf("cmd stdin 包装器读取数据失败: %s", err.Error())
// 			continue
// 		}

// 		if r.StreamEnd {
// 			s.stopFlag <- struct{}{}
// 			return
// 		}

// 		var lineData []byte
// 		if err = r.Data.Unmarshal(&lineData); err != nil {
// 			logs.Errorf("cmd stdin 包装器从数据包中解构数据内容失败: %s", err.Error())
// 			continue
// 		}

// 		s.w.Write(lineData)

// 	}

// }

// func (s *serverCmdStdinWrapper) WaitStop() {
// 	<-s.stopFlag
// 	s.lineBuf.Reset()
// 	s.lineBuf = nil
// }

// type serverCmdStderrWrapper struct {
// 	rw serverclientcommon.RWStreamInterface
// }

// func (s *serverCmdStderrWrapper) Write(p []byte) (n int, err error) {
// 	err = serverclientcommon.ErrSystemCall.ResultWithData("系统调用异常", p).WriteTo(s.rw)
// 	s.rw.Flush()
// 	return len(p), err
// }

// type serverCmdStdoutWrapper struct {
// 	rw serverclientcommon.RWStreamInterface
// }

// func (s *serverCmdStdoutWrapper) Write(p []byte) (n int, err error) {
// 	err = serverclientcommon.SuccessResult(p).WriteTo(s.rw)
// 	s.rw.Flush()
// 	return len(p), err
// }

var (
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

		// if callOption.BashName == "" {
		// 	return serverclientcommon.ErrValidation.Result("未正确配置当前Shell环境")
		// }

		// if callOption.WorkDir == "" {
		// 	return serverclientcommon.ErrValidation.Result("系统调用工作路径不能为空")
		// }

		// env := os.Environ()
		// if callOption.Env != nil {
		// 	env = append(env, callOption.Env...)
		// }

		// cmd := exec.Command(callOption.BashName, append(callOption.BaseArgs, callOption.Name)...)
		// cmd.Dir = callOption.WorkDir
		// cmd.Env = env
		// cmd.Stdout = &serverCmdStdoutWrapper{rw: rw}
		// cmd.Stderr = &serverCmdStderrWrapper{rw: rw}
		// cmd.Stdin = os.Stdin
		// // var cmdStdinWrapper *serverCmdStdinWrapper
		// // if wc, err := cmd.StdinPipe(); err == nil {
		// // 	cmdStdinWrapper = &serverCmdStdinWrapper{w: wc, rw: rw}
		// // 	go cmdStdinWrapper.Start()
		// // }

		// if err := cmd.Start(); err != nil {
		// 	return serverclientcommon.ErrSystemCall.Result("系统调用异常: " + err.Error())
		// }

		// if err := cmd.Wait(); err != nil {
		// 	return serverclientcommon.ErrSystemCall.Result("系统调用结果等待异常: " + err.Error())
		// }

		// if cmdStdinWrapper != nil {
		// 	serverclientcommon.ErrSystemCallEnd.Result("").WriteTo(rw)
		// 	cmdStdinWrapper.WaitStop()
		// }

		return serverclientcommon.SuccessResult(callOption)
	}

	// systemShellCurrentHandle serverclientcommon.CmdHandler = func(result *serverclientcommon.Result, conn serverclientcommon.RWStreamInterface) *serverclientcommon.Result {

	// }
)
