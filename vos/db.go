package vos

// DbPluginInfo 插件信息
type DbPluginInfo struct {
	// Id 主键
	Id string `json:"id,omitempty" gorm:"primary_key"`
	// Name 名称
	Name string `json:"name,omitempty"`
	// Desc 描述
	Desc string `json:"desc,omitempty"`
	// Icon 图标
	Icon string `json:"icon,omitempty"`
	// Path 路径
	Path string `json:"-"`
	// Md5 md5值 Hex格式
	Md5 string `json:"md5,omitempty"`
	// Sha1 sha1值 Hex格式
	Sha1 string `json:"sha1,omitempty"`
}
