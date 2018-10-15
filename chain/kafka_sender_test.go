package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func makeBlocks(chainInstance Chain, toBlockHeight uint64) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	if toBlockHeight <= latestSnapshotBlock.Height {
		return
	}

	count := toBlockHeight - latestSnapshotBlock.Height

	accountAddress1, _, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()
	for i := uint64(0); i < count; i++ {
		snapshotBlock, _ := newSnapshotBlock()
		chainInstance.InsertSnapshotBlock(snapshotBlock)

		for j := 0; j < 10; j++ {
			blocks, addressList, _ := randomSendViteBlock(snapshotBlock.Hash, &accountAddress1, &accountAddress2)
			chainInstance.InsertAccountBlocks(blocks)

			receiveBlocks, _ := newReceiveBlock(snapshotBlock.Hash, addressList[1], blocks[0].AccountBlock.Hash)
			chainInstance.InsertAccountBlocks(receiveBlocks)
		}

		if (i+1)%100 == 0 {
			fmt.Printf("Make %d snapshot blocks.\n", i+1)
		}

	}
}

func TestSend(t *testing.T) {
	chainInstance := getChainInstance()

	makeBlocks(chainInstance, 1000)
	//chainInstance.DeleteSnapshotBlocksToHeight(3)

	channel := make(chan int)
	<-channel
}
