package ssh

import (
	"fmt"
	"github.com/byzk-project-deploy/main-server/config"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

var (
	sshCmdCallHandle ssh.Handler = func(s ssh.Session) {
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
				_, _ = io.WriteString(s, "命令执行失败: "+err.Error())
				_ = s.Exit(1)
				return

			}
			go func() {
				for win := range winCh {
					setWinsize(f, win.Width, win.Height)
				}
			}()
			go func() {
				_, _ = io.Copy(f, s) // stdin
			}()
			_, _ = io.Copy(s, f) // stdout
			_ = cmd.Wait()
			_ = s.Exit(0)

		} else {
			_, _ = io.WriteString(s, "No PTY requested.\n")
			_ = s.Exit(1)
		}
	}
)
