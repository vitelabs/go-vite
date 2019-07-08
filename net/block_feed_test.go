package net

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func TestBlockFeed_Black(t *testing.T) {
	const black = "6771bc124fed97302328c13fb9a97919c8963b7b1f79a431091c7ace00ec28f4"

	feed := newBlockFeeder(make(map[types.Hash]struct{}))

	feed.SubscribeSnapshotBlock(func(block *ledger.SnapshotBlock, source types.BlockSource) {
		if block.Hash.String() == black {
			t.Fail()
		} else {
			fmt.Printf("%s/%d", block.Hash, block.Height)
		}
	})

	hash, _ := types.HexToHash(black)

	feed.notifySnapshotBlock(&ledger.SnapshotBlock{
		Hash: hash,
	}, types.RemoteCache)

	feed.notifySnapshotBlock(&ledger.SnapshotBlock{
		Hash: types.Hash{},
	}, types.RemoteCache)
}
