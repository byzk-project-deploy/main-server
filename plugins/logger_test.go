package plugins

import (
	"fmt"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLogger(t *testing.T) {
	a := assert.New(t)

	logPathDir := "testLog"
	defer os.RemoveAll(logPathDir)
	_ = os.MkdirAll(logPathDir, 0755)

	logger, err := NewLogger(&serverclientcommon.DbPluginInfo{
		Path: logPathDir,
	})
	if !a.NoError(err) {
		return
	}
	defer logger.Close()

	l := logger.PluginLogger()
	l.Debug("test debug...")
	l.Info("test info...")

	for i := 0; i < 10; i++ {
		l.Info(fmt.Sprintf("第%d次异步写入", i))
	}
	l.Warn("最后以警告结束吧")

}
