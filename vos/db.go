package vos

import (
	"encoding/json"
	"fmt"
	"github.com/byzk-project-deploy/main-server/security"
	serverclientcommon "github.com/byzk-project-deploy/server-client-common"
)

type ServerInfoContent []byte

func NewServerInfoContent(serverInfo *serverclientcommon.ServerInfo) (ServerInfoContent, error) {
	marshal, err := json.Marshal(serverInfo)
	if err != nil {
		return nil, fmt.Errorf("序列化服务器信息数据失败: %s", err.Error())
	}

	return security.Instance.DataEnvelope(marshal)
}

func (s ServerInfoContent) Unmarshal() (*serverclientcommon.ServerInfo, error) {
	rawData, err := security.Instance.DataUnEnvelope(s)
	if err != nil {
		return nil, err
	}

	var res *serverclientcommon.ServerInfo
	if err = json.Unmarshal(rawData, &res); err != nil {
		return nil, fmt.Errorf("反序列化服务器数据失败")
	}

	return res, nil
}

// DbServerInfo 服务器信息
type DbServerInfo struct {
	// Id 主键标识
	Id string `gorm:"primary_key"`
	// Name 名称
	Name string
	// Content 内容存放
	Content ServerInfoContent
}
