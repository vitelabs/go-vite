package chain_benchmark

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"time"
)

func loopInsertSnapshotBlock(chainInstance chain.Chain, interval time.Duration) chan struct{} {
	terminal := make(chan struct{})
	ticker := time.NewTicker(interval)

	createSbOptions := &test_tools.SnapshotOptions{
		MockTrie: false,
	}
	go func() {
		for {
			select {
			case <-terminal:
				ticker.Stop()
				return
			case <-ticker.C:
				sb := test_tools.CreateSnapshotBlock(chainInstance, createSbOptions)
				chainInstance.InsertSnapshotBlock(sb)
				fmt.Printf("latest snapshot height: %d. snapshot account number: %d\n", sb.Height, len(sb.SnapshotContent))
			}
		}
	}()
	return terminal
}
