package ssh

import (
	"github.com/gliderlabs/ssh"
	"github.com/patrickmn/go-cache"
	"time"
)

// passwordMap 密码map
var passwdCache = cache.New(1*time.Minute, 30*time.Second)

func AddPasswd(flag, password string) {
	passwdCache.SetDefault(flag, password)
}

var authOption = ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {
	s := ctx.User()
	if passwd, ok := passwdCache.Get(s); !ok || passwd != password {
		return false
	}
	passwdCache.Delete(s)
	return true
})
