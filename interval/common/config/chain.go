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

func (ch *Chain) Check(cfg *Base) error {
	if ch.StoreType == common.LevelDB {
		if len(ch.DBPath) == 0 {
			return errors.New("chain db path must be set")
		}
		ch.DataDir = cfg.DataDir
	}
	return nil
}
