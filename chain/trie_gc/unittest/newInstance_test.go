package trie_gc_unittest

import (
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

func newMarkerInstance(chainInstance chain.Chain) *trie_gc.Marker {
	return trie_gc.NewMarker(chainInstance, 0)

}
