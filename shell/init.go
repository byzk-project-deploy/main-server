package shell

import (
	"bufio"
	"bytes"
	"os"
	"strings"

	"github.com/byzk-project-deploy/main-server/config"
)

var (
	shellList []string

	currentShellCmdStr string
)

func init() {
	config.AddWatchAndNowExec(configWatch)
}

func configWatch(config *config.Info) {
	shellList = make([]string, 0, 18)
	shellConfig := config.Shell
	if len(shellConfig.AllowShellList) > 0 {
		shellList = append(shellList, shellConfig.AllowShellList...)
	}

	if shellConfig.AllowShellListFile == "" {
		return
	}

	if stat, err := os.Stat(shellConfig.AllowShellListFile); err != nil || stat.IsDir() {
		return
	}

	f, err := os.OpenFile(shellConfig.AllowShellListFile, os.O_RDONLY, 0655)
	if err != nil {
		return
	}
	defer f.Close()

	bufReader := bufio.NewReader(f)
	for {
		line, _, err := bufReader.ReadLine()
		if err != nil {
			return
		}
		line = bytes.TrimSpace(line)
		if line[0] == '#' || len(line) == 0 {
			continue
		}
		shellList = append(shellList, strings.TrimSpace(string(line)))
	}
}

func AllowList() []string {
	return shellList
}

func CurrentCmdStr() string {
	return currentShellCmdStr
}
