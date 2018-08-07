package impl

import (
	"github.com/vitelabs/go-vite/config"
	"os"
)

type CommonApisImpl struct {
}

func (CommonApisImpl) String() string {
	return "CommonApisImpl"
}

func (CommonApisImpl) LogDir(noop interface{}, reply *string) error {
	log.Info("LogDir")
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
