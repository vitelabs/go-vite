package test_tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	chain_test_tools "github.com/vitelabs/go-vite/v2/ledger/chain/test_tools"
	"github.com/vitelabs/go-vite/v2/vm/quota"
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

func NewTestChainInstance(dirName string, clear bool, genesis *config.Genesis) (chain.Chain, string) {
	tempDir := path.Join(chain_test_tools.DefaultDataDir(), dirName)
	fmt.Printf("tempDir: %s\n", tempDir)
	if clear {
		os.RemoveAll(tempDir)
	}

	quota.InitQuotaConfig(false, true)
	upgrade.CleanupUpgradeBox()

	genesisConfig := &config.Genesis{}
	if genesis != nil {
		genesisConfig = genesis
		upgrade.InitUpgradeBox(genesisConfig.UpgradeCfg.MakeUpgradeBox())
	} else {
		upgrade.InitUpgradeBox(upgrade.NewEmptyUpgradeBox().AddPoint(1, 10000000))
	}

	chainInstance := chain.NewChain(tempDir, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		panic(err)
	}
	if err := chainInstance.Start(); err != nil {
		panic(err)
	}
	return chainInstance, tempDir
}

func NewTestChainInstance2(dirName string, clear bool, genesisJson string) chain.Chain {
	var dataDir string

	if path.IsAbs(dirName) {
		dataDir = dirName
	} else {
		dataDir = path.Join(chain_test_tools.DefaultDataDir(), dirName)
	}

	if clear {
		os.RemoveAll(dataDir)
	}
	genesisConfig := &config.Genesis{}

	json.Unmarshal([]byte(genesisJson), genesisConfig)
	upgrade.InitUpgradeBox(config.MockGenesis().UpgradeCfg.MakeUpgradeBox())

	chainInstance := chain.NewChain(dataDir, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		panic(err)
	}
	// mock consensus
	//chainInstance.SetConsensus(&chain_test_tools.MockConsensus{})

	if err := chainInstance.Start(); err != nil {
		panic(err)
	}
	return chainInstance
}

func ClearChain(c chain.Chain, dir string) {
	if c != nil {
		c.Stop()
	}
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}
