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
	lock sync.Locker
	// startPluginMap 启动成功的插件
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

		pluginStatusInfo := &serverclientcommon.PluginStatusInfo{
			DbPluginInfo: pluginInfo,
		}
		m.pluginMap.Set(pluginInfo.Id, pluginStatusInfo)
	}

}

func (m *manager) InstallPlugin(pluginPath string) (res *serverclientcommon.DbPluginInfo, err error) {
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
	if err = os.Rename(targetFilePath, pluginSavePath); err != nil {
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

		m.pluginMap.Set(dbPluginInfo.Id, &serverclientcommon.PluginStatusInfo{
			DbPluginInfo: dbPluginInfo,
		})
		return nil
	})

}

func (m *manager) PluginStatusInfoList() (res []*serverclientcommon.PluginStatusInfo) {
	m.lock.Lock()
	defer m.lock.Unlock()
	keys := m.pluginMap.Keys()
	keyLen := len(keys)
	res = make([]*serverclientcommon.PluginStatusInfo, 0, keyLen)
	if keyLen == 0 {
		return
	}

	for i := range keys {
		key := keys[i]
		d, ok := m.pluginMap.Get(key)
		if !ok {
			continue
		}

		pluginStatusInfo, ok := d.(*serverclientcommon.PluginStatusInfo)
		if !ok {
			continue
		}

		res = append(res, pluginStatusInfo)
	}
	return
}

func (m *manager) FindOnePluginStatusInfoByIdOrName(idOrName string) (*serverclientcommon.PluginStatusInfo, error) {
	pluginList, err := m.pluginMatchNum(1, func(key string, val *serverclientcommon.PluginStatusInfo) bool {
		return idOrName == val.Id || idOrName == val.Name
	})
	if err != nil {
		return nil, err
	}
	return pluginList[0], nil
}

func (m *manager) FindPluginStatusInfoByIdOrName(idOrName string) (res []*serverclientcommon.PluginStatusInfo) {
	return m.pluginFilter(func(key string, val *serverclientcommon.PluginStatusInfo) bool {
		return strings.HasPrefix(val.Id, idOrName) || strings.HasPrefix(val.Name, idOrName)
	})
}

func (m *manager) pluginMatchNum(num int, fn func(key string, val *serverclientcommon.PluginStatusInfo) bool) (res []*serverclientcommon.PluginStatusInfo, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	keys := m.pluginMap.Keys()
	res = make([]*serverclientcommon.PluginStatusInfo, 0, len(keys))

	index := 0
	for i := range keys {
		key := keys[i]
		val, ok := m.pluginMap.Get(key)
		if !ok {
			continue
		}

		pluginInfo, ok := val.(*serverclientcommon.PluginStatusInfo)
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

func (m *manager) pluginFilter(fn func(key string, val *serverclientcommon.PluginStatusInfo) bool) (res []*serverclientcommon.PluginStatusInfo) {
	res, _ = m.pluginMatchNum(0, fn)
	return res
}
