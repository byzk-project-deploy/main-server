package config

import (
	"fmt"
	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
	"runtime"
)

var (
	currentOs   rpcinterfaces.OsOrArch
	currentArch rpcinterfaces.OsOrArch
)

func CurrentOs() rpcinterfaces.OsOrArch {
	return currentOs
}

func CurrentArch() rpcinterfaces.OsOrArch {
	return currentArch
}

func initOsAndArch() error {
	switch runtime.GOOS {
	case "linux":
		currentOs = rpcinterfaces.OsLinux
	case "darwin":
		currentOs = rpcinterfaces.OsDarwin
	default:
		return fmt.Errorf("未被支持的系统")
	}

	switch runtime.GOARCH {
	case "amd64":
		currentArch = rpcinterfaces.ArchAmd64
	case "arm":
		currentArch = rpcinterfaces.ArchArm64
	case "arm64":
		currentArch = rpcinterfaces.ArchArm64
	case "mips64le":
		currentArch = rpcinterfaces.ArchMips64le
	default:
		return fmt.Errorf("未被支持的架构")
	}
	return nil
}
