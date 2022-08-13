package config

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// Info 配置信息
type Info struct {
	// Listener 监听信息
	Listener *Listener `toml:"listener"`
	// Database 数据库配置
	Database *Database `toml:"database"`
	// Logs 日志配置
	Logs *LogConfig `toml:"logs"`
	// Shell shell配置
	Shell *ShellConfig `toml:"shell"`
	// Plugin 插件配置
	Plugin *PluginConfig `toml:"plugin"`
	// Remote 远程配置
	Remote *RemoteConfig `toml:"remote"`
	// Ssh ssh相关配置
	//Ssh *SSHConfig
}

type RemoteConfig struct {
	// InstallPath 安装路径, 默认为{{.homedir}}/.bypt/bin
	InstallPath string `toml:"installPath"`
}

// SSHConfig ssh配置
type SSHConfig struct {
	// ListenerPort 本地监听的端口, 默认为:0(随机端口)
	// 如果 Listener.AllowRemoteControl 设置为true本端口也将对外服务
	// 配置为固定是为了简化防火墙策略
	ListenerPort string `toml:"listenerPort"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	// StorePath 存放路径
	StorePath string `toml:"storePath"`
}

// Listener 监听配置
type Listener struct {
	// Allow 是否允许远程控制
	AllowRemoteControl bool `toml:"allowRemoteControl"`
	// Ip 监听ip地址
	Ip string `toml:"ip"`
	// Port 监听端口
	Port uint64 `toml:"port"`
	// SshAddr 本地监听的端口, 默认为:0(随机端口)
	// 如果 AllowRemoteControl 设置为true本端口也将对外服务
	// 配置为固定是为了简化防火墙策略
	SshAddr string `toml:"sshAddr"`
}

// Database 数据库配置
type Database struct {
	// Path 文件地址
	Path string `toml:"path"`
}

// LogLevelStr 日志级别字符串
type LogLevelStr string

// Level 日志级别
func (l LogLevelStr) Level() logrus.Level {
	switch strings.ToLower(string(l)) {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

// LogConfig 日志配置
type LogConfig struct {
	// Path 日志路径
	Path string `toml:"path"`
	// Level 日志等级
	Level LogLevelStr `toml:"level"`
}

// ShellConfig shell配置
type ShellConfig struct {
	// Current 当前shell
	Current string `toml:"current"`
	// Args 参数
	Args string `toml:"args"`
	// AllowShellList 允许的shell列表
	AllowShellList []string `toml:"allowShellList"`
	// AllowShellListFile 允许的shell列表文件，一行一个与 AllowShellList 并集
	AllowShellListFile string `toml:"allowShellListFile"`
}
