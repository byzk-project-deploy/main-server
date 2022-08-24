package plugins

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	"github.com/byzk-project-deploy/go-plugin"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var (
	// pluginInterfaceMap 插件接口实现
	pluginInterfaceMap = map[string]plugin.Plugin{
		rpcinterfaces.PluginNameBase: &rpcinterfaces.PluginBaseImpl{},
	}
	// pluginHandshakeConfig 插件握手协议
	pluginHandshakeConfig = rpcinterfaces.DefaultHandshakeConfig
)

type PluginStatusExtend struct {
	*serverclientcommon.PluginStatusInfo
	logger *Logger

	rpcCli       *plugin.Client
	rpcInterface rpcinterfaces.PluginBaseInterface

	exitChan    chan struct{}
	exitConfirm chan struct{}
	isStart     atomic.Bool
	lock        sync.Mutex
}

func (p *PluginStatusExtend) CheckStatus(status serverclientcommon.PluginStatus) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.Status == status
}

func (p *PluginStatusExtend) Close() {
	p.lock.Lock()
	if !p.isStart.Load() {
		return
	}
	close(p.exitChan)

	if p.rpcCli != nil {
		p.rpcCli.Kill()
		p.rpcCli = nil
	}
	p.lock.Unlock()

	<-p.exitConfirm
	close(p.exitConfirm)
}

func (p *PluginStatusExtend) Start() {
	if p.exitChan != nil {
		p.Close()
	}
	p.exitChan = make(chan struct{}, 1)
	p.exitConfirm = make(chan struct{}, 1)
	go func() {
		err := p.start()
		if err != nil {
			p.Status = serverclientcommon.PluginStatusNoRunning
			p.Msg = err.Error()
			go func() {
				<-p.exitChan
				p.exitConfirm <- struct{}{}
			}()
			p.Close()
			return
		}
		for {
			select {
			case <-p.exitChan:
				p.Status = serverclientcommon.PluginStatusNoRunning
				p.Msg = ""
				p.exitConfirm <- struct{}{}
				return
			default:
				if err = p.start(); err != nil {
					p.Status = serverclientcommon.PluginStatusErr
					p.Msg = err.Error()
				}
			}
		}
	}()
}

func (p *PluginStatusExtend) start() (err error) {
	p.lock.Lock()

	p.isStart.Store(true)
	p.Status = serverclientcommon.PluginStatusOk
	p.Msg = ""
	if p.rpcCli != nil {
		p.rpcCli.Kill()
		p.rpcCli = nil
	}

	sha512Sum, err := hex.DecodeString(p.Id)
	if err != nil {
		p.lock.Unlock()
		return fmt.Errorf("解析插件摘要失败: %s", err.Error())
	}

	p.rpcCli = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: *pluginHandshakeConfig,
		Plugins:         pluginInterfaceMap,
		Cmd:             exec.Command(filepath.Join(p.Path, "exec.plugin")),
		Logger:          p.logger.PluginLogger(),
		Stderr:          p.logger,
		SecureConfig: &plugin.SecureConfig{
			Checksum: sha512Sum,
			Hash:     sha512.New(),
		},
		TLSConfig: nil,
	})
	defer p.rpcCli.Kill()

	rpcClient, err := p.rpcCli.Client()
	if err != nil {
		p.lock.Unlock()
		return err
	}

	raw, err := rpcClient.Dispense(rpcinterfaces.PluginNameBase)
	if err != nil {
		p.lock.Unlock()
		return err
	}
	p.lock.Unlock()

	applicationPlugin, ok := raw.(pluginBaseExtendInterface)
	if !ok {
		return errors.New("插件接口不正确")
	}
	if err = applicationPlugin.Start(); err != nil {
		return err
	}

	applicationPlugin.Blocking("mainServer")
	applicationPlugin.Revoke("mainServer")
	return nil
}

type pluginBaseExtendInterface interface {
	rpcinterfaces.PluginBaseInterface
	rpcinterfaces.PluginBlocker
}
