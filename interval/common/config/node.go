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

func (nCfg *Node) Check() error {
	if nCfg.BaseCfg == nil {
		return errors.New("base config is empty")
	}
	if nCfg.P2pCfg == nil {
		return errors.New("p2p config is empty")
	}

	if nCfg.ConsensusCfg == nil {
		return errors.New("consensus config is empty")
	}

	if nCfg.ChainCfg == nil {
		return errors.New("chain config is empty")
	}

	e := nCfg.BaseCfg.Check()
	if e != nil {
		return e
	}

	e = nCfg.P2pCfg.Check(nCfg.BaseCfg)
	if e != nil {
		return e
	}

	e = nCfg.ConsensusCfg.Check(nCfg.BaseCfg)
	if e != nil {
		return e
	}

	e = nCfg.ChainCfg.Check(nCfg.BaseCfg)
	if e != nil {
		return e
	}

	if nCfg.MinerCfg != nil {
		e = nCfg.MinerCfg.Check(nCfg.BaseCfg)
		if e != nil {
			return e
		}
	}
	if nCfg.BootCfg != nil {
		e = nCfg.BootCfg.Check(nCfg.BaseCfg)
		if e != nil {
			return e
		}
	}

	return nil
}

func (nCfg *Node) Parse(cfg string) error {
	yamlFile, err := ioutil.ReadFile(cfg)
	if err != nil {
		log.Printf("cfg file err #%v ", err)
		return err
	}

	err = yaml.Unmarshal(yamlFile, nCfg)
	if err != nil {
		log.Fatalf("Unmarshal fail: %v", err)
		return err
	}
	return nil
}
