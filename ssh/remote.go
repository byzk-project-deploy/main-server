package ssh

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"net"
	"time"
)

type Remote struct {
	client *ssh.Client
}

func NewRemote(ip string, port uint16, username, password string) (*Remote, error) {
	config := &ssh.ClientConfig{
		User:    username,
		Timeout: 5 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	if port == 0 {
		port = 22
	}
	dial, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		return nil, err
	}

	return &Remote{
		client: dial,
	}, nil
}

func (r *Remote) Exec(command string) ([]byte, error) {
	session, err := r.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	return session.CombinedOutput(command)
}

func (r *Remote) Ftp() (*sftp.Client, error) {
	return sftp.NewClient(r.client)
}

func (r *Remote) FtpWrapper() (*SSHSftpWrapper, error) {
	ftp, err := r.Ftp()
	if err != nil {
		return nil, err
	}

	ftpWrapper, err := NewSSHSftpWrapper(ftp)
	if err != nil {
		return nil, err
	}

	ftpWrapper.s = r.client
	return ftpWrapper, nil
}

func (r *Remote) Close() {
	if r.client == nil {
		return
	}
	_ = r.client.Close()
	r.client = nil
}
