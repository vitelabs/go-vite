package trie_gc_unittest

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/trie_gc"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/node"
	"os"
	"path/filepath"
)

func newChainInstance(dirName string, clearDataDir bool) chain.Chain {
	dataDir := filepath.Join(node.DefaultDataDir(), dirName)

	if clearDataDir {
		os.RemoveAll(dataDir)
	}

	chainInstance := chain.NewChain(&config.Config{
		DataDir: dataDir,
	})
	chainInstance.Init()

	return chainInstance
}

func newMarkerInstance(chainInstance chain.Chain, dirName string, clear bool) (*trie_gc.Marker, error) {
	gcDir := filepath.Join(node.DefaultDataDir(), dirName, "ledger_gc")

	if clear {
		os.RemoveAll(gcDir)
	}

	gcDb, err := leveldb.OpenFile(gcDir, nil)
	if err != nil {
		return nil, err
	}
	mk, err := trie_gc.NewMarker(chainInstance, gcDb)
	if err != nil {
		return nil, err
	}
	return mk, nil

}
