package chain_genesis

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
)

type IndexDB interface {
	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedSubLedger map[types.Address][]*ledger.AccountBlock, sbLocation *chain_block.Location, abLocations []*chain_block.Location) error
}
type BlockDB interface {
}
