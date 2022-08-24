package plugins

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/byzk-project-deploy/main-server/sfnake"
	"github.com/byzk-project-deploy/main-server/vos"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	dbutils "github.com/byzk-worker/go-db-utils"
	"github.com/byzk-worker/go-db-utils/sqlite"
	"github.com/hashicorp/go-hclog"
	"github.com/jinzhu/gorm"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Logger struct {
	l                hclog.Logger
	dbClient         *sqlite.Client
	pipeReader       *io.PipeReader
	pipeWriter       *io.PipeWriter
	PluginInfo       *serverclientcommon.DbPluginInfo
	lock             *sync.Mutex
	exitChan         chan struct{}
	scanExitChan     chan struct{}
	haveNeedSyncData atomic.Bool
}

func NewLogger(pluginInfo *serverclientcommon.DbPluginInfo) (*Logger, error) {
	l := &Logger{PluginInfo: pluginInfo, lock: &sync.Mutex{}, exitChan: make(chan struct{}, 1), scanExitChan: make(chan struct{}, 1)}
	return l.init()
}

func (l *Logger) flushDb() {
	if !l.haveNeedSyncData.Load() {
		return
	}
	l.lock.Lock()
	defer l.lock.Unlock()
	_ = l.dbClient.Db().Transaction(func(tx *gorm.DB) error {
		return tx.Exec("INSERT OR REPLACE INTO filedb.db_plugin_logs SELECT * FROM db_plugin_logs;").Exec("DELETE FROM db_plugin_logs").Error
	})
	l.haveNeedSyncData.Store(false)
}

func (l *Logger) saveLog(pluginLog *vos.DbPluginLog) {
	id, _ := sfnake.GetID()
	pluginLog.Id = id
	l.lock.Lock()
	defer l.lock.Unlock()
	l.dbClient.Db().Exec("insert into db_plugin_logs (id, time, content, level) values (?, ?, ?, ?)", id, pluginLog.Time, pluginLog.Content, pluginLog.Level)
	l.haveNeedSyncData.Store(true)
}

func (l *Logger) Close() {
	if l.pipeWriter != nil {
		l.pipeWriter.Close()
	}

	if l.pipeReader != nil {
		l.pipeReader.Close()
	}

	<-l.scanExitChan
	l.exitChan <- struct{}{}
	<-l.exitChan
}

func (l *Logger) init() (*Logger, error) {
	l.l = hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     l,
		JSONFormat: true,
		TimeFormat: "20060102150405",
	})
	l.pipeReader, l.pipeWriter = io.Pipe()

	if l.PluginInfo == nil || l.PluginInfo.Path == "" {
		return nil, errors.New("缺失插件路径")
	}

	if stat, err := os.Stat(l.PluginInfo.Path); err != nil || !stat.IsDir() {
		return nil, errors.New("插件路径不存在")
	}

	dbFile := fmt.Sprintf("file:%s?auto_vacuum=1", filepath.Join(l.PluginInfo.Path, "log.db"))
	localDbClient := sqlite.New(fmt.Sprintf("file:%s?auto_vacuum=1", filepath.Join(l.PluginInfo.Path, "log.db")), dbutils.DefaultGetContextFn)
	if err := localDbClient.Init(); err != nil {
		return nil, fmt.Errorf("初始化本地日志库失败")
	}
	localDbClient.AutoMigrate(&vos.DbPluginLog{})
	localDbClient.Db().Close()

	l.dbClient = sqlite.New(fmt.Sprintf(":memory:"), dbutils.DefaultGetContextFn)
	if err := l.dbClient.Init(); err != nil {
		return nil, fmt.Errorf("打开日志库文件失败: %s", err.Error())
	}
	l.dbClient.AutoMigrate(&vos.DbPluginLog{})

	if err := l.dbClient.Db().Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS 'filedb';", dbFile)).Error; err != nil {
		return nil, fmt.Errorf("附加本地数据库失败: %s", err.Error())
	}

	go func() {
		for {
			c := time.After(10 * time.Second)
			select {
			case <-c:
				l.flushDb()
			case <-l.exitChan:
				l.flushDb()
				l.exitChan <- struct{}{}
				return
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(l.pipeReader)
		var logData map[string]string

		for scanner.Scan() {
			b := scanner.Bytes()
			if err := json.Unmarshal(b, &logData); err != nil {
				continue
			}
			l.saveLog(&vos.DbPluginLog{
				Time:    logData["@timestamp"],
				Content: logData["@message"],
				Level:   logData["@level"],
			})
		}
		l.scanExitChan <- struct{}{}
	}()

	return l, nil
}

func (l *Logger) PluginLogger() hclog.Logger {
	return l.l
}

func (l *Logger) Write(p []byte) (n int, err error) {
	return l.pipeWriter.Write(p)
}
