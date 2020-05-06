package config

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interval/common"
)

type Chain struct {
	StoreType common.StoreDBType `yaml:"storeType"`
	DBPath    string             `yaml:"dbPath"`
	DataDir   string
}

func (self *Chain) Check(cfg *Base) error {
	if self.StoreType == common.LevelDB {
		if len(self.DBPath) == 0 {
			return errors.New("chain db path must be set")
		}
		self.DataDir = cfg.DataDir
	}
	return nil
}
