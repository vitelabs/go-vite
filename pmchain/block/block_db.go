package chain_block

import "github.com/vitelabs/go-vite/ledger"

type BlockDB struct {
}

type SnapshotSegment struct {
	SnapshotBlock *ledger.SnapshotBlock
	AccountBlocks []*ledger.AccountBlock
}

func NewBlockDB() *BlockDB {
	return &BlockDB{}
}

func (bDB *BlockDB) Destroy() {}

func (bDB *BlockDB) Write(ss *SnapshotSegment) ([]*Location, *Location, error) {
	return nil, nil, nil
}

func (bDB *BlockDB) DeleteTo(location *Location) ([]*SnapshotSegment, []*ledger.AccountBlock, error) {
	return nil, nil, nil
}
