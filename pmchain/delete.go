package pmchain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) DeleteSnapshotBlocks(toHash *types.Hash) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	return nil, nil, nil
}

func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	return nil, nil, nil
}
