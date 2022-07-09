package user

import (
	"github.com/byzk-project-deploy/main-server/errors"
	"os/user"
)

var currentUser *user.User

func init() {
	u, err := user.Current()
	if err != nil {
		errors.ExitGetSysCurrentUser.Println("获取系统房前用户失败: %s", err.Error())
	}
	currentUser = u
}

// Current 获取当前用户
func Current() *user.User {
	return currentUser
}

// HomeDir 家目录
func HomeDir() string {
	return currentUser.HomeDir
}
