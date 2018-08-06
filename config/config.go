package config

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/common"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	P2P   `json:"P2P"`
	Miner `json:"Miner"`

	// global keys
	DataDir string `json:"DataDir"`
}

func (c Config) RunLogDir() string {
	return filepath.Join(c.DataDir, "runlog")
}

func (c Config) RunLogDirFile() (string, error) {
	filename := time.Now().Format("2006-01-02") + ".log"
	if err := os.MkdirAll(c.RunLogDir(), 0777); err != nil {
		return "", err
	}
	return filepath.Join(c.RunLogDir(), filename), nil

}

func (c Config) ConfigureLog() {
	if s, e := c.RunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.MultiHandler(
				log15.LvlFilterHandler(log15.LvlInfo, log15.StdoutHandler),
				log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(s, log15.TerminalFormat())),
			),
		)
	}
}

const configFileName = "vite.config.json"

var GlobalConfig *Config

func RecoverConfig() {
	GlobalConfig = &Config{
		P2P: P2P{
			Name:                 "vite-server",
			Sig:                  "",
			PrivateKey:           "",
			PublicKey:            "",
			MaxPeers:             100,
			MaxPassivePeersRatio: 2,
			MaxPendingPeers:      20,
			BootNodes:            nil,
			Addr:                 "0.0.0.0:8483",
			Datadir:              common.DefaultDataDir(),
			NetID:                3,
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
			fmt.Println("config file unmarshal error: ", err)
		}
	} else {
		fmt.Println("config file read error: ", err)
	}

	// set default value global keys
	if GlobalConfig.DataDir == "" {
		GlobalConfig.DataDir = common.DefaultDataDir()
	}

}
