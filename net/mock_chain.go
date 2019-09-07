package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

type mockChain struct {
	height uint64
}

func (mc mockChain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetSnapshotBlocks(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetAccountBlockByHeight(addr types.Address, height uint64) (*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetAccountBlocks(blockHash types.Hash, count uint64) ([]*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetAccountBlocksByHeight(addr types.Address, height uint64, count uint64) ([]*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mockChain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return &ledger.SnapshotBlock{
		Hash:   types.Hash{1, 1, 1},
		Height: mc.height,
	}
}

func (mc mockChain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return &ledger.SnapshotBlock{
		Hash:   types.Hash{1, 0, 0},
		Height: 1,
	}
}

func (mc mockChain) GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error) {
	panic("implement me")
}

func (mc mockChain) GetSyncCache() interfaces.SyncCache {
	panic("implement me")
}
