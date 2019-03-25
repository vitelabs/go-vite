package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
)

type mock_chain struct {
}

func (mc mock_chain) GetSubLedgerByHeight(start, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64) {
	panic("implement me")
}

func (mc mock_chain) GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64, error) {
	panic("implement me")
}

func (mc mock_chain) GetConfirmSubLedger(start, end uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetSnapshotBlocksByHash(origin *types.Hash, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetSnapshotBlocksByHeight(height, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error) {
	panic("implement me")
}

func (mc mock_chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return &ledger.SnapshotBlock{
		Hash:     types.Hash{},
		PrevHash: types.Hash{},
		Height:   1,
	}
}

func (mc mock_chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return &ledger.SnapshotBlock{
		Hash:     types.Hash{},
		PrevHash: types.Hash{},
		Height:   0,
	}
}

func (mc mock_chain) Compressor() *compress.Compressor {
	return nil
}
