package plugins

import (
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var (
	testPluginStatus = &PluginStatusExtend{
		PluginStatusInfo: &serverclientcommon.PluginStatusInfo{
			DbPluginInfo: &serverclientcommon.DbPluginInfo{
				Id:          "326ee2c141779a9ccb844c1ccc2d563ce3bfbb6d6e6157dfcc05a0d7044e4d3019b0f9b4036f208f1e9218a824468e6c7034e3102eb9b04194a63a3e6b27ec74",
				Name:        "test",
				ShortDesc:   "测试",
				Desc:        "测试插件",
				CreateTime:  time.Now(),
				Type:        rpcinterfaces.PluginTypeCmd,
				Path:        "testdatas/p",
				InstallTime: time.Now(),
			},
			Status:    serverclientcommon.PluginStatusNoRunning,
			Msg:       "",
			StartTime: time.Now(),
		},
	}
)

func init() {
	logger, err := NewLogger(testPluginStatus.DbPluginInfo)
	if err != nil {
		panic(err)
	}

	testPluginStatus.logger = logger
}

func TestPluginStart(t *testing.T) {
	a := assert.New(t)
	testPluginStatus.Start()
	time.Sleep(1 * time.Second)
	a.Equal(testPluginStatus.Status, serverclientcommon.PluginStatusOk)
	a.Equal(testPluginStatus.Msg, "")

	testPluginStatus.Close()
	a.Equal(testPluginStatus.Status, serverclientcommon.PluginStatusNoRunning)
	a.Equal(testPluginStatus.Msg, "")

}
