package config

import (
	"io/ioutil"
	"log"
	"encoding/json"
	"github.com/vitelabs/go-vite/common"
)

type Config struct {
	P2P				`json:"P2P"`
	Miner			`json:"Miner"`

	// global keys
	DataDir string	`json:"DataDir"`
}

const configFileName = "vite.config.json"

var GlobalConfig *Config

func RecoverConfig() {
	GlobalConfig =  &Config{
		P2P: P2P{
			Name:                 "vite-server",
			Sig:                  "",
			PrivateKey:           nil,
			PublicKey:            nil,
			MaxPeers:             100,
			MaxPassivePeersRatio: 2,
			MaxPendingPeers:      20,
			BootNodes:            nil,
			Addr:                 "0.0.0.0:8483",
			Datadir:              common.DefaultDataDir(),
			NetID:                2,
		},
		Miner: Miner{
			Miner:         false,
			Coinbase:      "",
			MinerInterval: 6,
		},
		DataDir: common.DefaultDataDir(),
	}
}

func init() {
	GlobalConfig = new(Config)

	if text, err := ioutil.ReadFile("vite.config.json"); err == nil {
		err = json.Unmarshal(text, GlobalConfig)
		if err != nil {
			log.Printf("config file unmarshal error: %v\n", err)
		}
	} else {
		log.Printf("config file read error: %v\n", err)
	}

	// set default value global keys
	if GlobalConfig.DataDir == "" {
		GlobalConfig.DataDir = common.DefaultDataDir()
	}
}
