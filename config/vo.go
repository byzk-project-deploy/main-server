package config

import (
	"github.com/sirupsen/logrus"
	"strings"
)

// Info 配置信息
type Info struct {
	// Listener 监听信息
	Listener *Listener
	// Database 数据库配置
	Database *Database
	// Logs 日志配置
	Logs *LogConfig
}

// Listener 监听配置
type Listener struct {
	// Allow 是否允许远程控制
	AllowRemoteControl bool
	// Ip 监听ip地址
	Ip string
	// Port 监听端口
	Port uint64
}

// Database 数据库配置
type Database struct {
	// Path 文件地址
	Path string
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
	Path string
	// Level 日志等级
	Level LogLevelStr
}
