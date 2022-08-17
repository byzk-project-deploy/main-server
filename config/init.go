package config

import (
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"os"
	"path/filepath"

	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/byzk-project-deploy/main-server/user"
	logs "github.com/byzk-worker/go-common-logs"
	"github.com/fsnotify/fsnotify"
)

type WatchRunFn func(config *Info)

const (
	BYPTRootName          = ".bypt"
	BYPTConfigFileName    = "config.toml"
	BYPTServerProgramName = "byptServer"
	BYPTClientProgramName = "bypt"
	BYPTServerVersion     = "3.0.0"
)

var (
	configDirPath  = filepath.Join(user.HomeDir(), BYPTRootName)
	configFilePath = filepath.Join(configDirPath, BYPTConfigFileName)
)

var (
	databaseFilePath = filepath.Join(configDirPath, "db", "local.data")
	logDirPath       = filepath.Join(configDirPath, "logs")
	pluginStorePath  = filepath.Join(configDirPath, "plugins")
)

var (
	configWatchRunList = make([]WatchRunFn, 0, 16)

	currentConfig = &Info{
		Listener: &Listener{
			AllowRemoteControl: false,
			Ip:                 "127.0.0.1",
			Port:               65526,
		},
		Database: &Database{
			Path: databaseFilePath,
		},
		Logs: &LogConfig{
			Path:  logDirPath,
			Level: "info",
		},
		Shell: &ShellConfig{
			Current:            "/usr/bin/sh",
			Args:               "-c",
			AllowShellListFile: "/etc/shells",
			AllowShellList:     []string{},
		},
		Plugin: &PluginConfig{
			StorePath: pluginStorePath,
		},
		Remote: &RemoteConfig{
			InstallPath: "{{.homedir}}/.bypt/bin",
		},
	}
)

func Current() *Info {
	return currentConfig
}

func Init() {
	err := initOsAndArch()
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("toml")

	stat, err := os.Stat(configFilePath)
	if err != nil || stat.IsDir() {
		if err = os.MkdirAll(configDirPath, 0755); err != nil {
			errors.ExitConfigFileCreatEmpty.Println("创建空的默认配置文件失败: %s", err.Error())
		}

		var f *os.File
		if f, err = os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY, 0655); err != nil {
			errors.ExitConfigFileCreatEmpty.Println("创建默认配置文件失败: %s", err.Error())
		}
		defer f.Close()

		if err = toml.NewEncoder(f).Encode(currentConfig); err != nil {
			errors.ExitConfigFileWriteToEmpty.Println("写出默认配置失败: %s", err.Error())
		}
	}

	if err = viper.ReadInConfig(); err != nil {
		errors.ExitConfigFileRead.Println("读取配置文件内容失败: %s", err.Error())
	}

	if err = viper.Unmarshal(&currentConfig); err != nil {
		errors.ExitConfigFileParser.Println("配置文件解析失败: %s", err.Error())
	}

	viper.OnConfigChange(onConfigChange)
	viper.WatchConfig()

}

// onConfigChange 配置文件发生改变
func onConfigChange(in fsnotify.Event) {
	if in.Op != fsnotify.Write {
		return
	}

	if err := viper.Unmarshal(&currentConfig); err != nil {
		logs.Warnf("配置文件已更新，但解析失败: %s", err.Error())
		return
	}

	for i := range configWatchRunList {
		fn := configWatchRunList[i]
		fn(currentConfig)
	}
}

func AddWatchFn(fn WatchRunFn) {
	configWatchRunList = append(configWatchRunList, fn)
}

func AddWatchAndNowExec(fn WatchRunFn) {
	fn(currentConfig)
	AddWatchFn(fn)
}
