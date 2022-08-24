package plugins

import (
	"crypto/sha512"
	"fmt"
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	packaging_plugin "github.com/byzk-project-deploy/packaging-plugin"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/byzk-worker/go-db-utils/sqlite"
	"github.com/go-base-lib/coderutils"
	"github.com/iancoleman/orderedmap"
	"github.com/jinzhu/gorm"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var m = &manager{
	lock: &sync.Mutex{},
}

type manager struct {
	lock *sync.Mutex
	// startPluginMap 已加载的插件集合
	pluginMap *orderedmap.OrderedMap
}

func (m *manager) init() {
	m.pluginMap = orderedmap.New()

	var pluginInfoList []*serverclientcommon.DbPluginInfo
	pluginModel := sqlite.Db().Model(&serverclientcommon.DbPluginInfo{})
	if err := pluginModel.Find(&pluginInfoList).Error; err != nil && err != gorm.ErrRecordNotFound {
		errors.ExitDatabaseQuery.Println("查询插件信息失败: %s", err.Error())
	}

	if len(pluginInfoList) == 0 {
		return
	}

	for i := range pluginInfoList {
		pluginInfo := pluginInfoList[i]

		logger, err := NewLogger(pluginInfo)
		if err != nil {
			errors.ExitLogDirCreate.Println("创建插件日志库失败: %s", err.Error())
		}
		pluginStatusInfo := &PluginStatusExtend{
			PluginStatusInfo: &serverclientcommon.PluginStatusInfo{
				DbPluginInfo: pluginInfo,
			},
			logger: logger,
		}
		m.pluginMap.Set(pluginInfo.Id, pluginStatusInfo)
		if pluginInfo.Enable {
			_ = m.Start(pluginStatusInfo.Id)
		}
	}
}

func (m *manager) Stop(idOrName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pluginStatusInfo, err := m.FindOneStatusInfoByIdOrName(idOrName)
	if err != nil {
		return fmt.Errorf("未识别的插件信息: %s", err.Error())
	}

	pluginStatusInfo.Enable = false

	if !pluginStatusInfo.CheckStatus(serverclientcommon.PluginStatusOk) {
		return nil
	}
	if err = sqlite.Db().Model(pluginStatusInfo.DbPluginInfo).Update(pluginStatusInfo.DbPluginInfo).Error; err != nil {
		return fmt.Errorf("更新插件启动信息失败: %s", err.Error())
	}
	pluginStatusInfo.StopTime = time.Now()
	pluginStatusInfo.Close()

	return nil

}

func (m *manager) Start(idOrName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pluginStatusInfo, err := m.FindOneStatusInfoByIdOrName(idOrName)
	if err != nil {
		return fmt.Errorf("未识别的插件信息: %s", err.Error())
	}

	if pluginStatusInfo.Path == "" {
		return fmt.Errorf("插件地址不存在: %s", err.Error())
	}

	if stat, err := os.Stat(filepath.Join(pluginStatusInfo.Path, "exec.plugin")); err != nil {
		return fmt.Errorf("获取插件文件的信息失败: %s", err.Error())
	} else if stat.IsDir() {
		return fmt.Errorf("插件文件不存在或已被篡改")
	}

	if pluginStatusInfo.CheckStatus(serverclientcommon.PluginStatusOk) {
		return nil
	}

	pluginStatusInfo.Enable = true
	if err = sqlite.Db().Model(&serverclientcommon.DbPluginInfo{}).Update(&pluginStatusInfo.DbPluginInfo).Error; err != nil {
		return fmt.Errorf("更新插件启动状态失败: %s", err.Error())
	}

	pluginStatusInfo.Start()
	return nil
}

func (m *manager) Uninstall(idOrName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	pluginStatusExtend, err := m.FindOneStatusInfoByIdOrName(idOrName)
	if err != nil {
		return err
	}

	if pluginStatusExtend.CheckStatus(serverclientcommon.PluginStatusOk) {
		return fmt.Errorf("请先停止插件后再尝试卸载")
	}

	return sqlite.Db().Transaction(func(tx *gorm.DB) error {
		pluginModel := tx.Model(&serverclientcommon.DbPluginInfo{})
		if err = pluginModel.Delete(&serverclientcommon.DbPluginInfo{
			Id: pluginStatusExtend.Id,
		}).Error; err != nil {
			return fmt.Errorf("删除插件信息失败: %s", err.Error())
		}

		os.RemoveAll(pluginStatusExtend.Path)
		m.pluginMap.Delete(pluginStatusExtend.Id)
		os.RemoveAll(pluginStatusExtend.Path)
		return nil
	})

}

func (m *manager) Install(pluginPath string) (res *serverclientcommon.DbPluginInfo, err error) {
	var (
		stat os.FileInfo

		pluginInfo     *rpcinterfaces.PluginInfo
		targetFilePath string
	)

	storePath := config.Current().Plugin.StorePath
	if stat, err = os.Stat(storePath); err != nil || !stat.IsDir() {
		if err = os.MkdirAll(storePath, 0755); err != nil {
			return nil, fmt.Errorf("创建插件存储目录失败: %s", err.Error())
		}
	}

	pluginInfo, targetFilePath, err = packaging_plugin.Unpacking(pluginPath, storePath)
	if err != nil {
		return nil, fmt.Errorf("插件安装失败: %s", err.Error())
	}
	defer func() {
		if err != nil || recover() != nil {
			_ = os.RemoveAll(targetFilePath)
		}
	}()

	sha512Sum, err := coderutils.HashByFilePath(sha512.New(), targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("获取插件Hash失败: %s", err.Error())
	}

	dbPluginInfo := &serverclientcommon.DbPluginInfo{
		Id:          sha512Sum.ToHexStr(),
		Author:      pluginInfo.Author,
		Name:        pluginInfo.Name,
		ShortDesc:   pluginInfo.ShortDesc,
		Desc:        pluginInfo.Desc,
		CreateTime:  pluginInfo.CreateTime,
		Type:        pluginInfo.Type,
		InstallTime: time.Now(),
	}

	pluginSavePath := filepath.Join(filepath.Dir(targetFilePath), dbPluginInfo.Id)
	_ = os.MkdirAll(pluginSavePath, 0755)
	if stat, err = os.Stat(pluginSavePath); err != nil {
		return nil, fmt.Errorf("创建插件存储目录失败: %s", err.Error())
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("获取插件存储路径失败: 非目录")
	}
	dbPluginInfo.Path = pluginSavePath

	_pluginPath := filepath.Join(pluginSavePath, "exec.plugin")
	if err = os.Rename(targetFilePath, _pluginPath); err != nil {
		return nil, fmt.Errorf("插件文件保存失败: %s", err.Error())
	}

	return dbPluginInfo, sqlite.Db().Transaction(func(tx *gorm.DB) error {
		dbPluginModel := tx.Model(&serverclientcommon.DbPluginInfo{})
		_ = dbPluginModel.Where("name=?", dbPluginInfo.Name).Delete(&serverclientcommon.DbPluginInfo{})
		if err = dbPluginModel.Save(&dbPluginInfo).Error; err != nil {
			return fmt.Errorf("保存插件信息失败: %s", err.Error())
		}
		m.lock.Lock()
		defer m.lock.Unlock()

		logger, err := NewLogger(dbPluginInfo)
		if err != nil {
			return fmt.Errorf("创建插件日志库失败: %s", err.Error())
		}
		m.pluginMap.Set(dbPluginInfo.Id, &PluginStatusExtend{
			PluginStatusInfo: &serverclientcommon.PluginStatusInfo{
				DbPluginInfo: dbPluginInfo,
			},
			logger: logger,
		})
		return nil
	})

}

func (m *manager) StatusInfoList() (res []*PluginStatusExtend) {
	m.lock.Lock()
	defer m.lock.Unlock()
	keys := m.pluginMap.Keys()
	keyLen := len(keys)
	res = make([]*PluginStatusExtend, 0, keyLen)
	if keyLen == 0 {
		return
	}

	for i := range keys {
		key := keys[i]
		d, ok := m.pluginMap.Get(key)
		if !ok {
			continue
		}

		pluginStatusInfo, ok := d.(*PluginStatusExtend)
		if !ok {
			continue
		}

		res = append(res, pluginStatusInfo)
	}
	return
}

func (m *manager) FindOneStatusInfoByIdOrName(idOrName string) (*PluginStatusExtend, error) {
	if m.lock.TryLock() {
		defer m.lock.Unlock()
	}
	pluginList, err := m.matchNum(1, func(key string, val *PluginStatusExtend) bool {
		return idOrName == val.Id || idOrName == val.Name
	})
	if err != nil {
		return nil, err
	}
	return pluginList[0], nil
}

func (m *manager) FindStatusInfoByIdOrName(idOrName string) (res []*PluginStatusExtend) {
	return m.filter(func(key string, val *PluginStatusExtend) bool {
		return strings.HasPrefix(val.Id, idOrName) || strings.HasPrefix(val.Name, idOrName)
	})
}

func (m *manager) matchNum(num int, fn func(key string, val *PluginStatusExtend) bool) (res []*PluginStatusExtend, err error) {
	if m.lock.TryLock() {
		defer m.lock.Unlock()
	}

	keys := m.pluginMap.Keys()
	res = make([]*PluginStatusExtend, 0, len(keys))

	index := 0
	for i := range keys {
		key := keys[i]
		val, ok := m.pluginMap.Get(key)
		if !ok {
			continue
		}

		pluginInfo, ok := val.(*PluginStatusExtend)
		if !ok {
			continue
		}

		if fn == nil {
			res = append(res, pluginInfo)
		} else if fn(key, pluginInfo) {
			res = append(res, pluginInfo)
		} else {
			continue
		}

		if num <= 0 {
			continue
		}

		index += 1
		if index >= num {
			return
		}
	}

	if num <= 0 {
		return
	}

	if index != num {
		return res, fmt.Errorf("不匹配的数量预期%d个,实际匹配到%d个", num, index)
	}

	return

}

func (m *manager) filter(fn func(key string, val *PluginStatusExtend) bool) (res []*PluginStatusExtend) {
	res, _ = m.matchNum(0, fn)
	return res
}
