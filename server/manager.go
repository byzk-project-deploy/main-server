package server

import (
	"bufio"
	"fmt"
	"github.com/akrennmair/slice"
	"github.com/byzk-project-deploy/main-server/config"
	"github.com/byzk-project-deploy/main-server/errors"
	"github.com/byzk-project-deploy/main-server/security"
	"github.com/byzk-project-deploy/main-server/ssh"
	"github.com/byzk-project-deploy/main-server/vos"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
	"github.com/byzk-worker/go-db-utils/sqlite"
	transportstream "github.com/go-base-lib/transport-stream"
	"github.com/jinzhu/gorm"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/sftp"
	"github.com/tjfoc/gmsm/gmtls"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type remoteServerConn struct {
	s     *serverclientcommon.ServerInfo
	conn  net.Conn
	err   error
	bufRW *bufio.ReadWriter
}

func (r *remoteServerConn) ConnToStream() (*transportstream.Stream, error) {
Start:
	if r.err == nil {
		return nil, r.err
	}

	if r.conn == nil {
		r.err = r.Connection()
		goto Start
	}

	stream := transportstream.NewStream(r.bufRW)

	if _, err := serverclientcommon.CmdHello.Exchange(stream); err != nil {
		if err == io.EOF {
			r.Close()
			goto Start
		}
		r.err = err
		return nil, err
	}

	return stream, nil
}

func (r *remoteServerConn) UpdateServerInfo(s *serverclientcommon.ServerInfo) {
	r.Close()
	r.s = s
}

// Connection 连接远端服务，连接失败返回错误
func (r *remoteServerConn) Connection() (err error) {
	defer func() {
		r.err = err
	}()

	serverInfo := r.s
	if serverInfo.Status < serverclientcommon.ServerStatusNetworkErr {
		return fmt.Errorf("当前服务的状态无法进行远程连接")
	}

	if serverInfo.IP == nil {
		return fmt.Errorf("服务器IP地址不能为空")
	}

	if serverInfo.Port == 0 {
		return fmt.Errorf("缺失端口信息")
	}

	r.Close()

	tlsConfig, err := security.Instance.GetTlsClientConfig(security.GetRemoteServerName(serverInfo.IP), security.LinkRemoteDnsFlag, serverInfo.IP)
	if err != nil {
		return fmt.Errorf("创建连接凭据失败: %s", err.Error())
	}

	if r.conn, err = gmtls.Dial("tcp", fmt.Sprintf("%s:%d", serverInfo.IP.String(), serverInfo.Port), tlsConfig); err != nil {
		r.conn = nil
		return fmt.Errorf("%s", err.Error())
	}
	r.bufRW = bufio.NewReadWriter(bufio.NewReader(r.conn), bufio.NewWriter(r.conn))

	return nil
}

// Close 关闭连接
func (r *remoteServerConn) Close() {
	r.err = nil
	r.bufRW = nil
	if r.conn != nil {
		r.conn.Close()
	}
}

// Error 返回当前连接中的异常信息
func (r *remoteServerConn) Error() error {
	return r.err
}

func newRemoteServerConn(s *serverclientcommon.ServerInfo) *remoteServerConn {
	return &remoteServerConn{
		s: s,
	}
}

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

func init() {
	var dbServerInfoList []*vos.DbServerInfo
	if err := sqlite.Db().Model(&vos.DbServerInfo{}).Find(&dbServerInfoList).Error; err != nil && err != gorm.ErrRecordNotFound {
		errors.ExitDatabaseQuery.Println("查询服务器信息失败, 请检查数据库相关配置: %s", err.Error())
	}

	dbLen := len(dbServerInfoList)
	remoteServerList = make([]*serverclientcommon.ServerInfo, 0, dbLen)
	remoteServerMap = make(map[string]*serverclientcommon.ServerInfo, dbLen)
	remoteServerConnMap = make(map[string]*remoteServerConn, dbLen)
	remoteAliasMap = make(map[string]*serverclientcommon.ServerInfo, dbLen)

	for i := range dbServerInfoList {
		dbServerInfo := dbServerInfoList[i]
		serverInfo, err := dbServerInfo.Content.Unmarshal()
		if err != nil {
			errors.ExitDatabaseQuery.Println("服务器数据解密失败, 数据可能已被篡改, 请检查相关数据项: %s", err.Error())
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

		remoteServerConnMap[serverInfo.Id] = newRemoteServerConn(serverInfo)
	}

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
		remoteConnErr := remoteServerConnMap[serverInfo.Id].Connection()
		if remoteConnErr != nil {
			serverInfo.Status = serverclientcommon.ServerStatusNetworkErr
			serverInfo.EndMsg = remoteConnErr.Error()
		} else {
			serverInfo.Status = serverclientcommon.ServerRunning
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

	return nil
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

//func Add(serverInfo *serverclientcommon.ServerInfo) error {
//	if serverInfo.IP == nil {
//		return fmt.Errorf("服务器IP不能为空")
//	}
//
//	ipStr := serverInfo.IP.String()
//	if remoteServerMap[ipStr] != nil {
//		return fmt.Errorf("服务器已经存在，请勿重新添加")
//	}
//
//	if serverInfo.Port <= 0 {
//		return fmt.Errorf("服务器端口不能为空")
//	}
//
//	certRes, err := security.Instance.GeneratorClientPemCert(security.GetRemoteServerName(serverInfo.IP), security.LinkRemoteDnsFlag, serverInfo.IP)
//	if err != nil {
//		return fmt.Errorf("服务器证书生成失败: %s", certRes)
//	}
//	serverInfo.ClientCertPem = certRes.CertPem
//	serverInfo.ClientPrivatePem = certRes.PrivateKeyPem
//	serverInfo.JoinTime = time.Now()
//
//	idStr, err := sfnake.GetIdStr()
//	if err != nil {
//		return fmt.Errorf("获取ID失败: %s", err.Error())
//	}
//
//	content, err := vos.NewServerInfoContent(serverInfo)
//	if err != nil {
//		return fmt.Errorf("序列化服务器信息失败: %s", err.Error())
//	}
//
//	return sqlite.Db().Transaction(func(tx *gorm.DB) error {
//		dbServerInfoModel := tx.Model(&vos.DbServerInfo{})
//
//		count := 0
//		if err = dbServerInfoModel.Where(&vos.DbServerInfo{
//			Name: ipStr,
//		}).Count(&count).Error; err != nil {
//			return fmt.Errorf("查询服务器数量失败: %s", err.Error())
//		}
//
//		if count > 0 {
//			return fmt.Errorf("服务器已存在")
//		}
//
//		saveServerInfo := &vos.DbServerInfo{
//			Id:      idStr,
//			Name:    ipStr,
//			Content: content,
//		}
//
//		if err = dbServerInfoModel.Save(&saveServerInfo).Error; err != nil {
//			return fmt.Errorf("保存服务器信息失败: %s", err.Error())
//		}
//
//		if serverPort, err := checkAndInstallBypt(ipStr, serverInfo.SSHPort, serverInfo.SSHUser, serverInfo.SSHPassword); err != nil {
//			return err
//		} else if serverPort != serverInfo.Port {
//			serverInfo.Port = serverPort
//			content, err = vos.NewServerInfoContent(serverInfo)
//			if err != nil {
//				return fmt.Errorf("序列化服务器信息失败: %s", err.Error())
//			}
//			saveServerInfo.Content = content
//			if err = dbServerInfoModel.Update(&saveServerInfo).Error; err != nil {
//				return fmt.Errorf("更新服务器端口信息失败: %s", err.Error())
//			}
//		}
//		return nil
//	})
//}

//func checkAndInstallBypt(ip string, port int, username, password string) (p int, err error) {
//	var (
//		byptServerConfigBytes []byte
//		buf                   *bytes.Buffer
//	)
//
//	remoteCmd, err := ssh.NewRemote(ip, port, username, password)
//	if err != nil {
//		return 0, err
//	}
//
//	ftp, err := remoteCmd.Ftp()
//	if err != nil {
//		return 0, err
//	}
//	defer ftp.Close()
//
//	pwdPath, err := ftp.Getwd()
//	if err != nil {
//		return 0, fmt.Errorf("获取服务器[%s]当前目录失败: %s", ip, err.Error())
//	}
//
//	tmlData := map[string]any{"homedir": pwdPath, "ip": ip}
//
//	remoteByptDir := filepath.Join(pwdPath, ".bypt")
//	remoteConfigPath := filepath.Join(remoteByptDir, "config.toml")
//	remoteConfigFile, err := ftp.Open(remoteConfigPath)
//	if err == nil {
//		goto Success
//	}
//	defer func() {
//		if err != nil {
//			_ = ftp.Remove(remoteConfigPath)
//			_ = ftp.RemoveDirectory(remoteByptDir)
//		}
//	}()
//	defer remoteConfigFile.Close()
//
//	_ = ftp.MkdirAll(remoteByptDir)
//	if remoteConfigFile, err = ftp.Create(remoteConfigPath); err != nil {
//		return 0, fmt.Errorf("在远程服务器[%s]中创建配置文件失败: %s", ip, err.Error())
//	}
//
//	if buf, err = execTemplateToBuf(remoteConfigTml, tmlData); err != nil {
//		return 0, fmt.Errorf("生成服务器[%s]对应的配置失败: %s", ip, err.Error())
//	}
//
//	if _, err = io.Copy(remoteConfigFile, buf); err != nil {
//		return 0, fmt.Errorf("配置文件上传失败: %s", err.Error())
//	}
//
//	if _, err = remoteConfigFile.Seek(0, 0); err != nil {
//		return 0, fmt.Errorf("移动文件指针失败: %s", err.Error())
//	}
//
//Success:
//	byptServerConfigBytes, err = io.ReadAll(remoteConfigFile)
//	if err != nil {
//		return 0, fmt.Errorf("读取服务器[%s]配置失败: %s", ip, err.Error())
//	}
//
//	v := viper.New()
//	v.SetConfigType("toml")
//	if err = v.ReadConfig(bytes.NewReader(byptServerConfigBytes)); err != nil {
//		return 0, fmt.Errorf("解析远端服务配置失败, 请检查服务器[%s]上的配置文件内容", ip)
//	}
//
//	remoteAllowControl := v.GetBool("listener.allowRemoteControl")
//	if !remoteAllowControl {
//		return 0, fmt.Errorf("远程服务[%s]不允许进行控制", ip)
//	}
//
//	remotePortStr := v.GetString("listener.port")
//	if remotePortStr == "" {
//		return 0, fmt.Errorf("远程服务[%s]没有正在监听的端口，请检查相关配置", ip)
//	}
//
//	remotePort, err := strconv.Atoi(remotePortStr)
//	if err != nil {
//		return 0, fmt.Errorf("转换远程服务[%s]监听端口失败, 请检查服务器内的配置", ip)
//	}
//
//	return remotePort, nil
//}
