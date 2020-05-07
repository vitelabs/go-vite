package config

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interval/common"
)

type Miner struct {
	Enabled     bool   `yaml:"enabled"`
	HexCoinbase string `yaml:"coinbase"`
}

func (mCfg *Miner) CoinBase() common.Address {
	coinbase := common.HexToAddress(mCfg.HexCoinbase)
	return coinbase
}
func (mCfg *Miner) Check(cfg *Base) error {
	if !mCfg.Enabled {
		return nil
	}
	if len(mCfg.CoinBase().String()) == 0 {
		return errors.New("miner coinbase must be set when miner enabled")
	}
	return nil
}
