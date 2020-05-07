package config

import "github.com/pkg/errors"

type P2P struct {
	NodeId       string `yaml:"nodeId"`
	Port         int    `yaml:"port"`
	NetId        int    `yaml:"netId"`
	LinkBootAddr string `yaml:"bootAddr"`
}

func (ppCfg *P2P) Check(cfg *Base) error {
	if len(ppCfg.NodeId) == 0 {
		return errors.New("p2p node Id must be set")
	}

	if len(ppCfg.LinkBootAddr) == 0 {
		return errors.New("p2p Link Boot Addr must be set")
	}
	return nil
}

type Boot struct {
	BootAddr string `yaml:"addr"`
	Enabled  bool   `yaml:"enabled"`
}

func (bCfg *Boot) Check(cfg *Base) error {
	if !bCfg.Enabled {
		return nil
	}

	if len(bCfg.BootAddr) == 0 {
		return errors.New("boot node addr must be set")
	}
	return nil
}
