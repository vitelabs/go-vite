package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
	"time"
)

func makeBlocks(chainInstance Chain, toBlockHeight uint64) {
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	if toBlockHeight <= latestSnapshotBlock.Height {
		return
	}

	count := toBlockHeight - latestSnapshotBlock.Height

	var addrList [][2]types.Address
	for i := 0; i < 1000; i++ {
		oneAddr := [2]types.Address{}
		oneAddr[0], _, _ = types.CreateAddress()
		oneAddr[1], _, _ = types.CreateAddress()
		addrList = append(addrList, oneAddr)
	}

	for i := uint64(0); i < count; i++ {
		snapshotBlock, _ := newSnapshotBlock()
		chainInstance.InsertSnapshotBlock(snapshotBlock)

		for j := 0; j < 10; j++ {
			for _, addr := range addrList {
				blocks, _, _ := randomSendViteBlock(snapshotBlock.Hash, &addr[0], &addr[1])
				chainInstance.InsertAccountBlocks(blocks)

				receiveBlocks, _ := newReceiveBlock(snapshotBlock.Hash, addr[1], blocks[0].AccountBlock.Hash)
				chainInstance.InsertAccountBlocks(receiveBlocks)
			}
		}

		if (i+1)%10 == 0 {
			fmt.Printf("Make %d snapshot blocks.\n", i+1)
		}

	}
}

func TestSend(t *testing.T) {
	chainInstance := getChainInstance()
	go func() {
		for {
			time.Sleep(time.Second)
			for index, p := range chainInstance.KafkaSender().RunProducers() {
				lastId, _ := chainInstance.GetLatestBlockEventId()
				fmt.Printf("%d %d %d\n", index, p.HasSend(), lastId)
			}
		}

	}()

	//makeBlocks(chainInstance, 1000)
	//chainInstance.DeleteSnapshotBlocksToHeight(3)

	channel := make(chan int)
	<-channel
}
