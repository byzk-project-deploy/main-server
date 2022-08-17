package server

import (
	"fmt"
	"github.com/akrennmair/slice"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/byzk-project-deploy/main-server/ssh"
	"github.com/byzk-project-deploy/main-server/vos"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/byzk-worker/go-db-utils/sqlite"
	"github.com/jinzhu/gorm"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/sftp"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RangeFileFn func(filename string, relativePath string, filesize int64, reader io.Reader) error

var (
	// remoteServerMap ip:server
	remoteServerMap map[string]*serverclientcommon.ServerInfo
	//remoteServerConnMap 远程服务连接Map
	remoteServerConnMap map[string]*remoteServerConn
	// remoteAliasMap alias:server
	remoteAliasMap map[string]*serverclientcommon.ServerInfo
	// remoteServerList sortServer
	remoteServerList []*serverclientcommon.ServerInfo
)

func initManager() {
	var dbServerInfoList []*vos.DbServerInfo
	if err := sqlite.Db().Model(&vos.DbServerInfo{}).Find(&dbServerInfoList).Error; err != nil && err != gorm.ErrRecordNotFound {
		errors.ExitDatabaseQuery.Println("查询服务器信息失败, 请检查数据库相关配置: %s", err.Error())
	}

	dbLen := len(dbServerInfoList)
	remoteServerList = make([]*serverclientcommon.ServerInfo, 0, dbLen)
	remoteServerMap = make(map[string]*serverclientcommon.ServerInfo, dbLen)
	remoteServerConnMap = make(map[string]*remoteServerConn, dbLen)
	remoteAliasMap = make(map[string]*serverclientcommon.ServerInfo, dbLen)

	w := &sync.WaitGroup{}
	for i := range dbServerInfoList {
		dbServerInfo := dbServerInfoList[i]
		serverInfo, err := dbServerInfo.Content.Unmarshal()
		if err != nil {
			errors.ExitDatabaseQuery.Println("服务器数据解密失败, 数据可能已被篡改, 请检查相关数据项: %s", err.Error())
		}

		if serverInfo.Status == serverclientcommon.ServerRunning {
			serverInfo.Status = serverclientcommon.ServerStatusNoRun
		}

		remoteServerConnMap[serverInfo.Id] = newRemoteServerConn(serverInfo)
		if serverInfo.Status >= serverclientcommon.ServerStatusNetworkErr {
			w.Add(1)
			go func(s *serverclientcommon.ServerInfo) {
				if err = remoteServerConnMap[s.Id].Ping(); err == nil {
					s.Status = serverclientcommon.ServerRunning
					s.EndMsg = ""
				} else {
					s.Status = serverclientcommon.ServerStatusNetworkErr
					s.EndMsg = err.Error()
				}
				w.Done()
			}(serverInfo)

		}
		remoteServerList = append(remoteServerList, serverInfo)

		serverInfo.Id = dbServerInfo.Id
		remoteServerMap[serverInfo.IP.String()] = serverInfo
		aliasLen := len(serverInfo.Alias)
		if aliasLen == 0 {
			continue
		}

		for j := range serverInfo.Alias {
			alias := serverInfo.Alias[j]
			remoteAliasMap[alias] = serverInfo
		}

	}
	w.Wait()
}

var Manager = &manager{
	lock: &sync.Mutex{},
}

type manager struct {
	lock *sync.Mutex
}

func (m *manager) addServerInfoToStore(serverInfo *serverclientcommon.ServerInfo) (res *serverclientcommon.ServerInfo, err error) {
	if m.lock.TryLock() {
		defer m.lock.Unlock()
	}

	var (
		dbServerInfoContent vos.ServerInfoContent

		ok bool
	)
	if res, ok = remoteServerMap[serverInfo.Id]; ok {
		return
	}

	serverInfo.JoinTime = time.Now()
	serverInfo.Status = serverclientcommon.ServerStatusNoCheck
	dbServerModel := sqlite.Db().Model(&vos.DbServerInfo{})
	dbServerInfoContent, err = vos.NewServerInfoContent(serverInfo)
	if err != nil {
		err = serverclientcommon.ErrServerInside.Newf("序列化服务器数据失败: %s", err.Error())
		return
	}

	if err = dbServerModel.Save(&vos.DbServerInfo{
		Id:      serverInfo.Id,
		Name:    serverInfo.Id,
		Content: dbServerInfoContent,
	}).Error; err != nil {
		err = serverclientcommon.ErrServerInside.Newf("保存服务器信息失败: %s", err.Error())
		return
	}
	remoteServerMap[serverInfo.Id] = serverInfo
	remoteServerConnMap[serverInfo.Id] = newRemoteServerConn(serverInfo)
	remoteServerList = append(remoteServerList, serverInfo)
	return serverInfo, nil
}

func (m *manager) updateServerInfoToStore(serverInfo *serverclientcommon.ServerInfo) (err error) {
	if m.lock.TryLock() {
		defer m.lock.Unlock()
	}

	nowServerIndex := -1
	nowServerInfo, ok := remoteServerMap[serverInfo.Id]
	if !ok {
		return serverclientcommon.ErrServerInside.New("要修改的服务器信息不存在")
	} else {
		for i := range remoteServerList {
			if remoteServerList[i].Id == nowServerInfo.Id {
				nowServerIndex = i
			}
		}
	}

	defer func() {
		if err != nil {
			remoteServerConnMap[serverInfo.Id].UpdateServerInfo(nowServerInfo)
		}
	}()

	if nowServerIndex == -1 {
		return serverclientcommon.ErrServerInside.New("获取服务器索引失败")
	}

	var dbServerInfoContent vos.ServerInfoContent
	dbServerInfoContent, err = vos.NewServerInfoContent(serverInfo)
	if err != nil {
		err = serverclientcommon.ErrServerInside.Newf("序列化服务器数据失败: %s", err.Error())
		return
	}

	remoteServerConnMap[serverInfo.Id].UpdateServerInfo(serverInfo)
	if serverInfo.Status >= serverclientcommon.ServerStatusNetworkErr {
		remoteServerConnMap[serverInfo.Id].Reconnection()
		err = remoteServerConnMap[serverInfo.Id].Ping()
		if err != nil {
			serverInfo.Status = serverclientcommon.ServerStatusNoRun
			serverInfo.EndMsg = err.Error()
		} else {
			serverInfo.Status = serverclientcommon.ServerRunning
			serverInfo.EndMsg = ""
		}
	}

	if err = sqlite.Db().Model(&vos.DbServerInfo{}).Where("id=?", serverInfo.Id).Update(&vos.DbServerInfo{Content: dbServerInfoContent}).Error; err != nil {
		err = serverclientcommon.ErrServerInside.Newf("更新服务器信息失败: %s", err.Error())
		return
	}

	if len(nowServerInfo.Alias) > 0 {
		for i := range nowServerInfo.Alias {
			alias := nowServerInfo.Alias[i]
			delete(remoteAliasMap, alias)
		}
	}

	if len(serverInfo.Alias) > 0 {
		for i := range serverInfo.Alias {
			alias := serverInfo.Alias[i]
			remoteServerMap[alias] = serverInfo
		}
	}

	remoteServerMap[serverInfo.Id] = serverInfo
	remoteServerList[nowServerIndex] = serverInfo

	return
}

func (m *manager) clearAliasByIp(ip string) error {
	serverInfo, ok := remoteServerMap[ip]
	if !ok {
		return fmt.Errorf("获取服务器[%s]信息失败", ip)
	}

	rawServerAlias := serverInfo.Alias

	serverInfo.Alias = nil

	if err := m.updateServerInfoToStore(serverInfo); err != nil {
		return err
	}

	for i := range rawServerAlias {
		alias := rawServerAlias[i]
		delete(remoteAliasMap, alias)
	}
	return nil
}

func (m *manager) GetServerFtp(serverInfo *serverclientcommon.ServerInfo) (*ssh.SSHSftpWrapper, error) {
	if m.lock.TryLock() {
		defer m.lock.Unlock()
	}
	cli, err := ssh.NewRemote(serverInfo.Id, serverInfo.SSHPort, serverInfo.SSHUser, serverInfo.SSHPassword)
	if err != nil {
		return nil, fmt.Errorf("连接服务器[%s]失败: %s", serverInfo.Id, err.Error())
	}

	wrapper, err := cli.FtpWrapper()
	if err != nil {
		cli.Close()
		return nil, err
	}

	return wrapper, nil
}

func (m *manager) repair(s *serverclientcommon.ServerInfo) (res *serverclientcommon.RemoteServerRepairResInfo) {
	var (
		err error

		port   int
		status serverclientcommon.ServerStatus
	)

	serverConn := remoteServerConnMap[s.Id]

	res = &serverclientcommon.RemoteServerRepairResInfo{
		Ip:      s.Id,
		Success: true,
	}
	defer func() {
		if err != nil {
			res.Success = false
			res.ErrMsg = err.Error()
			return
		}
	}()

	if s.Status == serverclientcommon.ServerRunning {
		if err = serverConn.Ping(); err == nil {
			return
		}
	}

	if port, status, err = m.CheckRemoteServer(s.IP, s.SSHPort, s.SSHUser, s.SSHPassword); err != nil {
		return
	}

	if status == serverclientcommon.ServerStatusNeedInstall {
		err = fmt.Errorf("bypt主程序未安装, 暂不支持自动安装，请前往服务器中手动安装之后再尝试手动修复")
		return
	}

	s.Port = port
	s.Status = status
	if status >= serverclientcommon.ServerStatusNetworkErr {
		if err = m.updateServerInfoToStore(s); err != nil {
			s.Status = serverclientcommon.ServerStatusNetworkErr
		} else if s.EndMsg != "" {
			err = fmt.Errorf(s.EndMsg)
			s.Status = serverclientcommon.ServerStatusNetworkErr
		}
	}

	return
}

func (m *manager) Repair() (res []*serverclientcommon.RemoteServerRepairResInfo) {
	m.lock.Lock()
	defer m.lock.Unlock()

	serverListLen := len(remoteServerMap)
	if serverListLen == 0 {
		return nil
	}

	res = make([]*serverclientcommon.RemoteServerRepairResInfo, 0, serverListLen)

	w := &sync.WaitGroup{}

	for k := range remoteServerMap {
		serverInfo := remoteServerMap[k]
		w.Add(1)
		go func() {
			defer w.Done()
			res = append(res, m.repair(serverInfo))
		}()
	}

	w.Wait()

	return
}

func (m *manager) SettingAlias(aliasList []string, ip string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	serverInfo, ok := remoteServerMap[ip]
	if !ok {
		return fmt.Errorf("获取服务器[%s]信息失败", ip)
	}

	if len(aliasList) == 0 {
		return m.clearAliasByIp(ip)
	}

	var info *serverclientcommon.ServerInfo
	for i := range aliasList {
		alias := aliasList[i]
		info, ok = remoteAliasMap[alias]
		if ok && info.Id != ip {
			return fmt.Errorf("别名[%s]已被服务器[%s]占用", alias, info.Id)
		}
	}

	serverInfo.Alias = aliasList
	return m.updateServerInfoToStore(serverInfo)
}

func (m *manager) UpdateServerInfo(serverInfo *serverclientcommon.ServerInfo) error {
	return m.updateServerInfoToStore(serverInfo)
}

func (m *manager) List(searchName []string) []*serverclientcommon.ServerInfo {
	m.lock.Lock()
	defer m.lock.Unlock()

	if len(searchName) == 0 {
		return remoteServerList
	}

	return slice.Filter(remoteServerList, func(s *serverclientcommon.ServerInfo) bool {
		id := s.Id
		alias := strings.Join(s.Alias, ",")

		for i := range searchName {
			searchKeyword := searchName[i]
			if strings.Contains(id, searchKeyword) || strings.Contains(alias, searchKeyword) {
				return true
			}
		}

		return false
	})
}

func (m *manager) FindByServerName(serverName string) (*serverclientcommon.ServerInfo, error) {
	if m.lock.TryLock() {
		defer m.lock.Unlock()
	}

	serverInfo, ok := remoteServerMap[serverName]
	if ok {
		return serverInfo, nil
	}

	serverInfo, ok = remoteAliasMap[serverName]
	if ok {
		return serverInfo, nil
	}
	return nil, fmt.Errorf("未获取到[%s]对应的服务器信息", serverName)
}

func (m *manager) RemoveServer(serverName string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if serverName == "" {
		return nil
	}

	serverInfo, err := m.FindByServerName(serverName)
	if err != nil {
		return err
	}

	return sqlite.Db().Transaction(func(tx *gorm.DB) error {
		if err = tx.Model(&vos.DbServerInfo{}).Where("id=?", serverInfo.Id).Delete(&vos.DbServerInfo{}).Error; err != nil {
			return serverclientcommon.ErrServerInside.Newf("删除服务器信息失败: %s", err.Error())
		}

		conn, ok := remoteServerConnMap[serverInfo.Id]
		if ok {
			conn.Close()
		}

		for i := range remoteServerList {
			if remoteServerList[i].Id == serverInfo.Id {
				remoteServerList = append(remoteServerList[:i], remoteServerList[i+1:]...)
				break
			}
		}

		for i := range serverInfo.Alias {
			alias := serverInfo.Alias[i]
			delete(remoteAliasMap, alias)
		}

		delete(remoteServerMap, serverInfo.Id)
		delete(remoteServerConnMap, serverInfo.Id)
		return nil
	})

}

// AddServer 添加一个服务器
func (m *manager) AddServer(serverInfo *serverclientcommon.ServerInfo) (resStatus serverclientcommon.ServerStatus, err error) {
	if serverInfo, err = m.addServerInfoToStore(serverInfo); err != nil {
		return
	}
	defer func() {
		if err != nil {
			serverInfo.EndMsg = err.Error()
		}
		serverInfo.Status = resStatus
		_ = m.updateServerInfoToStore(serverInfo)
		err = nil
	}()

	serverInfo.Port, resStatus, err = m.CheckRemoteServer(serverInfo.IP, serverInfo.SSHPort, serverInfo.SSHUser, serverInfo.SSHPassword)
	return
}

// CheckRemoteServer 检查远程服务是否已经成功安装
func (m *manager) CheckRemoteServer(ip net.IP, sshPort uint16, username, password string) (targetPort int, status serverclientcommon.ServerStatus, err error) {
	ipStr := ip.String()

	var (
		remoteServer *ssh.Remote

		remoteFtp *sftp.Client

		remoteHomeDir            string
		remoteByptRootDir        string
		remoteByptConfigFilePath string
		remoteByptConfigFileStat os.FileInfo
		remoteByptConfigFile     *sftp.File

		remoteConfig *config.Info
	)
	if remoteServer, err = ssh.NewRemote(ipStr, sshPort, username, password); err != nil {
		if strings.Contains(err.Error(), "password") {
			status = serverclientcommon.ServerStatusUserErr
			err = fmt.Errorf("用户名或密码验证失败")
		} else {
			status = serverclientcommon.ServerStatusCheckErr
		}
		return
	}

	if remoteFtp, err = remoteServer.Ftp(); err != nil {
		return
	}

	if remoteHomeDir, err = remoteFtp.Getwd(); err != nil {
		return
	}

	remoteByptRootDir = filepath.Join(remoteHomeDir, config.BYPTRootName)
	remoteByptConfigFilePath = filepath.Join(remoteByptRootDir, config.BYPTConfigFileName)

	if remoteByptConfigFileStat, err = remoteFtp.Stat(remoteByptConfigFilePath); err != nil || remoteByptConfigFileStat.IsDir() {
		err = nil
		status = serverclientcommon.ServerStatusNeedInstall
		return
	}

	if remoteByptConfigFile, err = remoteFtp.OpenFile(remoteByptConfigFilePath, os.O_RDONLY); err != nil {
		return
	}
	defer remoteByptConfigFile.Close()

	if err = toml.NewDecoder(remoteByptConfigFile).Decode(&remoteConfig); err != nil {
		err = fmt.Errorf("解析服务器配置文件失败, 请检查服务器中配置文件[%s]的内容是否正确", ipStr)
		return
	}

	if !config.IsDev() {
		var output []byte
		if output, err = remoteServer.Exec("byptServer --test"); err != nil || !strings.HasSuffix(string(output), "\n"+config.BYPTServerVersion) {
			return 0, serverclientcommon.ServerStatusNeedInstall, nil
		}
	}

	targetPort = int(remoteConfig.Listener.Port)
	status = serverclientcommon.ServerStatusNoRun
	return
}

func (m *manager) ReadFileInfo(serverName string, path string, t serverclientcommon.UploadType) (os.FileInfo, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	serverInfo, err := m.FindByServerName(serverName)
	if err != nil {
		return nil, err
	}

	switch t {
	case serverclientcommon.UploadTypeSSHFtp:
		serverFtp, err := m.GetServerFtp(serverInfo)
		if err != nil {
			return nil, err
		}
		defer serverFtp.Close()

		path = serverFtp.PathReplace(path)
		stat, err := serverFtp.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("打开服务器[%s]中的[%s]路径失败: %s", serverInfo.Id, path, err.Error())
		}

		return stat, nil
	default:
		return nil, fmt.Errorf("无法匹配的协议类型")
	}

}

func (m *manager) rangeFileWithSSHSftp(ftpWrapper *ssh.SSHSftpWrapper, path string, relativePath string, fn RangeFileFn) error {
	stat, err := ftpWrapper.Stat(path)
	if err != nil {
		return fmt.Errorf("获取路径[%s]的文件信息失败: %s", path, err.Error())
	}

	if stat.IsDir() {
		dirChildren, err := ftpWrapper.ReadDir(path)
		if err != nil {
			return fmt.Errorf("获取目录[%s]的子文件失败: %s", path, err.Error())
		}
		for i := range dirChildren {
			child := dirChildren[i]
			if err = m.rangeFileWithSSHSftp(ftpWrapper, filepath.Join(path, child.Name()), filepath.Join(relativePath, child.Name()), fn); err != nil {
				return err
			}
		}
		return nil
	}

	f, err := ftpWrapper.OpenFile(path, os.O_RDONLY)
	if err != nil {
		return fmt.Errorf("打开文件[%s]失败: %s", path, err.Error())
	}
	defer f.Close()
	return fn(f.Name(), relativePath, stat.Size(), f)
}

func (m *manager) RangeFile(serverName string, path string, t serverclientcommon.UploadType, fn RangeFileFn) error {
	serverInfo, err := m.FindByServerName(serverName)
	if err != nil {
		return err
	}

	switch t {
	case serverclientcommon.UploadTypeSSHFtp:
		ftpWrapper, err := m.GetServerFtp(serverInfo)
		if err != nil {
			return err
		}
		defer ftpWrapper.Close()
		path = ftpWrapper.PathReplace(path)
		return m.rangeFileWithSSHSftp(ftpWrapper, path, "", fn)
	default:
		return fmt.Errorf("无法匹配的协议类型")
	}
}
