package db

import (
	"fmt"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	dbutils "github.com/byzk-worker/go-db-utils"
	"github.com/byzk-worker/go-db-utils/sqlite"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"os"
	"path/filepath"
)

var dbFile string

func init() {
	config.AddWatchAndNowExec(configWatch)
}

func configWatch(config *config.Info) {
	if dbFile == config.Database.Path {
		return
	} else {
		dbFile = config.Database.Path
	}

	dbDir := filepath.Dir(dbFile)
	if stat, err := os.Stat(dbDir); err != nil || !stat.IsDir() {
		_ = os.MkdirAll(dbDir, 0755)
	}

	sqlite.Close()

	if err := sqlite.Init(fmt.Sprintf("file:%s?auto_vacuum=1", dbFile), dbutils.DefaultGetContextFn); err != nil {
		errors.ExitDatabaseOpen.Println("打开数据库文件[%s]失败: %s", dbFile, err.Error())
	}

	sqlite.EnableDebug()

	initDataTable()

}

func initDataTable() {
	sqlite.Db().Exec("PRAGMA auto_vacuum = 1;")
	if err := sqlite.Db().
		AutoMigrate(&serverclientcommon.DbPluginInfo{}).
		Error; err != nil {
		errors.ExitDatabaseCreateTable.Println("创建数据库表失败, 错误信息: %s", err.Error())
	}
	//sqlite.Db().AutoMigrate(&vos.DbVersionInfo{})
	//go startExec()
}
