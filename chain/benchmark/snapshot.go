package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

type snapshotOptions struct {
	mockTrie bool
}

func createSnapshotBlock(chainInstance Chain, options *snapshotOptions) *ledger.SnapshotBlock {
	latestBlock := chainInstance.GetLatestSnapshotBlock()
	now := time.Now()
	snapshotBlock := &ledger.SnapshotBlock{
		Height:    latestBlock.Height + 1,
		PrevHash:  latestBlock.Hash,
		Timestamp: &now,
	}

	content := chainInstance.GetNeedSnapshotContent()
	snapshotBlock.SnapshotContent = content

	if options != nil && options.mockTrie {
		snapshotBlock.StateHash = types.Hash{}
	} else {
		trie, _ := chainInstance.GenStateTrie(latestBlock.StateHash, content)
		snapshotBlock.StateTrie = trie
		snapshotBlock.StateHash = *trie.Hash()
	}

	snapshotBlock.Hash = snapshotBlock.ComputeHash()

	return snapshotBlock
}

func loopInsertSnapshotBlock(chainInstance Chain, interval time.Duration) chan struct{} {
	terminal := make(chan struct{})
	ticker := time.NewTicker(interval)

	createSbOptions := &snapshotOptions{
		mockTrie: false,
	}
	go func() {
		for {
			select {
			case <-terminal:
				ticker.Stop()
				return
			case <-ticker.C:
				sb := createSnapshotBlock(chainInstance, createSbOptions)
				chainInstance.InsertSnapshotBlock(sb)
				fmt.Printf("latest snapshot height: %d. snapshot account number: %d\n", sb.Height, len(sb.SnapshotContent))
			}
		}
	}()
	return terminal
}
