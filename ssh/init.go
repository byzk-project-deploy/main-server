package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	logs "github.com/byzk-worker/go-common-logs"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/patrickmn/go-cache"
)

// passwordMap 密码map
var passwdCache = cache.New(1*time.Minute, 30*time.Second)

var sshListenerPort string

func AddPasswd(flag, password string) {
	passwdCache.SetDefault(flag, password)
}

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func init() {
	ssh.Handle(func(s ssh.Session) {
		cmdArgList := s.Command()
		cmdAndArgsStr := strings.Join(cmdArgList, " ")
		shellConfig := config.Current().Shell
		shellName := strings.TrimSpace(shellConfig.Current)
		shellArgs := strings.Split(strings.TrimSpace(shellConfig.Args), " ")
		shellArgs = append(shellArgs, cmdAndArgsStr)

		cmd := exec.Command(shellName, shellArgs...)
		ptyReq, winCh, isPty := s.Pty()

		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, "SHELL="+shellName)
		systemCallOption, err := serverclientcommon.SystemCallCommandRunOptionUnmarshal(s)
		if err == nil {
			if cmd.Env != nil {
				cmd.Env = append(cmd.Env, systemCallOption.Env...)
			}
			if systemCallOption.WorkDir != "" {
				cmd.Dir = systemCallOption.WorkDir
			}
		}

		if isPty {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))

			f, err := pty.Start(cmd)
			if err != nil {
				io.WriteString(s, "命令执行失败: "+err.Error())
			}
			go func() {
				for win := range winCh {
					setWinsize(f, win.Width, win.Height)
				}
			}()
			go func() {
				io.Copy(f, s) // stdin
			}()
			io.Copy(s, f) // stdout
			cmd.Wait()
			s.Exit(0)

		} else {
			io.WriteString(s, "No PTY requested.\n")
			s.Exit(1)
		}
	})

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		errors.ExitSSHServerListener.Println("监听命令转发服务失败: " + err.Error())
	}

	sshListenerPort = l.Addr().String()
	i := strings.LastIndexByte(sshListenerPort, ':')
	if i >= 0 {
		sshListenerPort = sshListenerPort[i:]
	}

	logs.Infof("system command server listener %s", sshListenerPort)
	go func() {
		if err = ssh.Serve(l, nil, ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {
			s := ctx.User()
			if passwd, ok := passwdCache.Get(s); !ok || passwd != password {
				return false
			}
			passwdCache.Delete(s)
			return true
		})); err != nil {
			logs.Errorf("命令转发服务异常停止，请尝试重新启动: %s", err.Error())
			errors.ExitSSHServerListenerExit.Exit()
		}
		errors.ExitSSHServerListenerExit.Exit()
	}()
}

func ListenerPortStr() string {
	return sshListenerPort
}
