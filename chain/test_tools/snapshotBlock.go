package test_tools

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

type SnapshotOptions struct {
	MockTrie bool
}

func CreateSnapshotBlock(chainInstance chain.Chain, options *SnapshotOptions) *ledger.SnapshotBlock {
	latestBlock := chainInstance.GetLatestSnapshotBlock()
	now := time.Now()
	snapshotBlock := &ledger.SnapshotBlock{
		Height:    latestBlock.Height + 1,
		PrevHash:  latestBlock.Hash,
		Timestamp: &now,
	}

	content := chainInstance.GetNeedSnapshotContent()
	snapshotBlock.SnapshotContent = content

	if options != nil && options.MockTrie {
		snapshotBlock.StateHash = types.Hash{}
	} else {
		trie, _ := chainInstance.GenStateTrie(latestBlock.StateHash, content)
		snapshotBlock.StateTrie = trie
		snapshotBlock.StateHash = *trie.Hash()
	}

	snapshotBlock.Hash = snapshotBlock.ComputeHash()

	return snapshotBlock
}
