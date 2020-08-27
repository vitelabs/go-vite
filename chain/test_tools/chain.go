package test_tools

import (
	"encoding/json"
	"os"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vm/quota"
)

func NewChainInstanceFromDir(dirName string, clear bool, genesis string) (chain.Chain, error) {
	if clear {
		os.RemoveAll(dirName)
	}
	quota.InitQuotaConfig(false, true)
	genesisConfig := &config.Genesis{}
	json.Unmarshal([]byte(genesis), genesisConfig)

	chainInstance := chain.NewChain(dirName, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		return nil, err
	}
	chainInstance.Start()
	return chainInstance, nil
}
