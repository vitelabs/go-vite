package config

import (
	"io/ioutil"
	"log"
	"encoding/json"
	"github.com/vitelabs/go-vite/common"
	"os"
)

type Config struct {
	P2P				`json:"P2P"`
	Miner			`json:"Miner"`

	// global keys
	DataDir string	`json:"DataDir"`
}

const configFileName = "vite.config.json"

var GlobalConfig *Config

func init() {
	GlobalConfig = new(Config)

	_, err := os.Stat(configFileName)
	if err != nil {
		log.Printf("config file %s is not exist\n", configFileName)
		return
	}

	text, err := ioutil.ReadFile("vite.config.json")
	if err != nil {
		log.Printf("config file read error: %v\n", err)
		return
	}

	err = json.Unmarshal(text, GlobalConfig)

	if err != nil {
		log.Printf("config file unmarshal error: %v\n", err)
	}

	// set default value global keys
	if GlobalConfig.DataDir == "" {
		GlobalConfig.DataDir = common.DefaultDataDir()
	}
}
