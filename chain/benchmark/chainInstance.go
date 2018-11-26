package chain_benchmark

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/node"
	"os"
	"path/filepath"
)

func newChainInstance(dirName string, clearDataDir bool) chain.Chain {
	return newChainInstanceByDataDir("benchmark", dirName, clearDataDir)
}

func newTestChainInstance() chain.Chain {
	return newChainInstanceByDataDir("testdata", "", false)
}

func newChainInstanceByDataDir(dataRoot, dirName string, clearDataDir bool) chain.Chain {
	dataDir := filepath.Join(node.DefaultDataDir(), dataRoot, dirName)

	if clearDataDir {
		os.RemoveAll(dataDir)
	}

	chainInstance := chain.NewChain(&config.Config{
		DataDir: dataDir,
	})
	chainInstance.Init()
	chainInstance.Start()

	return chainInstance
}
