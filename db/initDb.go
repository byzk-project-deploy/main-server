package db

import (
	"fmt"
	"github.com/byzk-worker/go-common-logs"
	dbutils "github.com/byzk-worker/go-db-utils"
	"github.com/byzk-worker/go-db-utils/sqlite"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"os"
	"path/filepath"
)

func InitDb() {
	dataDir := filepath.Join(".d")
	_ = os.MkdirAll(dataDir, 0777)

	dbFile := filepath.Join(dataDir, "local.data")

	if err := sqlite.Init(fmt.Sprintf("file:%s?auto_vacuum=1", dbFile), dbutils.DefaultGetContextFn); err != nil {
		logs.Errorln("打开数据库文件失败: ", err.Error())
		os.Exit(2)
	}

	initDataTable()
}

func initDataTable() {
	sqlite.Db().Exec("PRAGMA auto_vacuum = 1;")
	//sqlite.Db().AutoMigrate(&vos.DbVersionInfo{})
	//go startExec()
}
