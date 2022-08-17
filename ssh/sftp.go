package ssh

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"path/filepath"
	"strings"
)

type SSHSftpWrapper struct {
	*sftp.Client
	s       *ssh.Client
	workdir string
	homeDir string
}

func NewSSHSftpWrapper(c *sftp.Client) (*SSHSftpWrapper, error) {
	s := &SSHSftpWrapper{
		Client: c,
	}

	homedir, err := c.RealPath("")
	if err != nil {
		return nil, fmt.Errorf("获取用户家目录失败: %s", err.Error())
	}

	workdir, err := c.Getwd()
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("获取工作路径失败: %s", err.Error())
	}

	s.homeDir = homedir
	s.workdir = workdir
	return s, nil
}

func (s *SSHSftpWrapper) Workdir() string {
	return s.workdir
}

func (s *SSHSftpWrapper) PathReplace(p string) string {
	p = strings.ReplaceAll(p, "~", s.workdir)
	p = strings.ReplaceAll(p, "./", "")
	if p[0] != '/' {
		p = filepath.Join(s.workdir, p)
	}
	return p
}

func (s *SSHSftpWrapper) Close() {
	if s.Client != nil {
		s.Client.Close()
	}

	if s.s != nil {
		s.s.Close()
	}
}
