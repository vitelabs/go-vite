package chain_block

import (
	"github.com/vitelabs/go-vite/ledger"
)

const (
	LocationSize = 12
)

type Location [LocationSize]byte

func (bDB *BlockDB) Write(accountBlocks []*ledger.AccountBlock, snapshotBlock *ledger.SnapshotBlock) ([]*Location, *Location, error) {
	return nil, nil, nil
}
