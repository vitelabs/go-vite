package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type hotData struct {
	ds *dataSet

	latestSnapshotBlock *ledger.SnapshotBlock

	latestAccountBlocks map[types.Address]types.Hash
}

func newHotData(ds *dataSet) *hotData {
	return &hotData{
		ds:                  ds,
		latestAccountBlocks: make(map[types.Address]types.Hash),
	}
}

func (hd *hotData) SetLatestSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) {
	hd.latestSnapshotBlock = snapshotBlock
}

func (hd *hotData) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return hd.latestSnapshotBlock
}

func (hd *hotData) InsertAccountBlock(block *ledger.AccountBlock) {
	hd.latestAccountBlocks[block.AccountAddress] = block.Hash
}

func (hd *hotData) DeleteAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		delete(hd.latestAccountBlocks, block.AccountAddress)
	}
}

func (hd *hotData) GetLatestAccountBlock(addr types.Address) *ledger.AccountBlock {
	hash, ok := hd.latestAccountBlocks[addr]
	if !ok {
		return nil
	}

	return hd.ds.GetAccountBlock(hash)
}
