package main

import (
	"encoding/json"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
)

func newChain(dirName string) (chain.Chain, error) {
	genesisConfig := &config.Genesis{}
	err := json.Unmarshal([]byte(genesis), genesisConfig)
	if err != nil {
		return nil, err
	}

	chainInstance := chain.NewChain(dirName, &config.Chain{}, genesisConfig)

	if err = chainInstance.Init(); err != nil {
		return nil, err
	}

	err = chainInstance.Start()
	if err != nil {
		return nil, err
	}

	return chainInstance, nil
}
