package config

import "os"

var isDev bool

func IsDev() bool {
	return isDev
}

func init() {
	isDev = os.Getenv("DEV") != ""
}
