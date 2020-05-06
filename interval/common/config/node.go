package config

import (
	"errors"
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type Node struct {
	P2pCfg       *P2P       `yaml:"p2p"`
	ConsensusCfg *Consensus `yaml:"consensus"`
	MinerCfg     *Miner     `yaml:"miner"`
	ChainCfg     *Chain     `yaml:"chain"`
	BaseCfg      *Base      `yaml:"base"`
	BootCfg      *Boot      `yaml:"boot"`
}

func (self *Node) Check() error {
	if self.BaseCfg == nil {
		return errors.New("base config is empty")
	}
	if self.P2pCfg == nil {
		return errors.New("p2p config is empty")
	}

	if self.ConsensusCfg == nil {
		return errors.New("consensus config is empty")
	}

	if self.ChainCfg == nil {
		return errors.New("chain config is empty")
	}

	e := self.BaseCfg.Check()
	if e != nil {
		return e
	}

	e = self.P2pCfg.Check(self.BaseCfg)
	if e != nil {
		return e
	}

	e = self.ConsensusCfg.Check(self.BaseCfg)
	if e != nil {
		return e
	}

	e = self.ChainCfg.Check(self.BaseCfg)
	if e != nil {
		return e
	}

	if self.MinerCfg != nil {
		e = self.MinerCfg.Check(self.BaseCfg)
		if e != nil {
			return e
		}
	}
	if self.BootCfg != nil {
		e = self.BootCfg.Check(self.BaseCfg)
		if e != nil {
			return e
		}
	}

	return nil
}

func (self *Node) Parse(cfg string) error {
	yamlFile, err := ioutil.ReadFile(cfg)
	if err != nil {
		log.Printf("cfg file err #%v ", err)
		return err
	}

	err = yaml.Unmarshal(yamlFile, self)
	if err != nil {
		log.Fatalf("Unmarshal fail: %v", err)
		return err
	}
	return nil
}
