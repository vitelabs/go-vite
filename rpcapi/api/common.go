package api

import (
	"github.com/vitelabs/go-vite/config"
	"os"
)

type CommonApi struct {
}

func (CommonApi) String() string {
	return "CommonApi"
}

func (CommonApi) LogDir() string {
	log.Info("CommonApi LogDir")
	info, e := os.Stat(config.GlobalConfig.RunLogDir())
	if e != nil {
		return ""
	}
	if !info.IsDir() {
		return ""
	}
	return config.GlobalConfig.RunLogDir()
}

