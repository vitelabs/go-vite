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

func (CommonApi) LogDir(noop interface{}, reply *string) error {
	log.Info("CommonApi LogDir")
	info, e := os.Stat(config.GlobalConfig.RunLogDir())
	if e != nil {
		*reply = ""
		return nil
	}
	if !info.IsDir() {
		*reply = ""
		return nil
	}
	*reply = config.GlobalConfig.RunLogDir()
	return nil
}

