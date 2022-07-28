package config

import (
	"os"
	"path/filepath"

	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/byzk-project-deploy/main-server/user"
	logs "github.com/byzk-worker/go-common-logs"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type WatchRunFn func(config *Info)

var (
	configDirPath  = filepath.Join(user.HomeDir(), ".bypt")
	configFilePath = filepath.Join(configDirPath, "config.toml")
)

var (
	databaseFilePath = filepath.Join(configDirPath, "db", "local.data")
	logDirPath       = filepath.Join(configDirPath, "logs")
)

var (
	configWatchRunList = make([]WatchRunFn, 0, 16)

	currentConfig *Info
)

func Current() *Info {
	return currentConfig
}

func init() {
	err := initOsAndArch()
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("toml")
	viper.SetDefault("listener.allowRemoteControl", false)
	viper.SetDefault("listener.ip", "127.0.0.1")
	viper.SetDefault("listener.port", "65526")
	viper.SetDefault("database.path", databaseFilePath)
	viper.SetDefault("logs.level", "info")
	viper.SetDefault("logs.path", logDirPath)
	viper.SetDefault("shell.current", "/usr/bin/sh")
	viper.SetDefault("shell.args", "-i -c")
	viper.SetDefault("shell.allowShellListFile", "/etc/shells")
	viper.SetDefault("shell.allowShellList", []string{})

	stat, err := os.Stat(configFilePath)
	if err != nil || stat.IsDir() {
		if err = os.MkdirAll(configDirPath, 0755); err != nil {
			errors.ExitConfigFileCreatEmpty.Println("创建空的默认配置文件失败: %s", err.Error())
		}

		if err = viper.WriteConfigAs(configFilePath); err != nil {
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
