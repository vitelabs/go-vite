package config

import "github.com/vitelabs/go-vite/v2/common/upgrade"

type Upgrade struct {
	Level  string
	Points map[string]*upgrade.UpgradePoint
}

func (cfg *Upgrade) MakeUpgradeBox() upgrade.UpgradeBox {
	if cfg == nil {
		panic("unknown upgrade nil")
	}
	if cfg.Level == "mainnet" {
		return upgrade.NewMainnetUpgradeBox()
	} else if cfg.Level == "latest" {
		return upgrade.NewLatestUpgradeBox()
	} else if cfg.Level == "custom" {
		return upgrade.NewCustomUpgradeBox(cfg.Points)
	}
	panic("unknown upgrade level")
}
