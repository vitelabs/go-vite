package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (iDB *IndexDB) GetSnapshotBlockLocationByHash(hash *types.Hash) (*chain_block.Location, error) {
	return nil, nil
}

func (iDB *IndexDB) GetSnapshotBlockLocation(height uint64) (*chain_block.Location, error) {
	return nil, nil
}

func (iDB *IndexDB) GetLatestSnapshotBlockHeight() (uint64, error) {
	return 1, nil
}

func (iDB *IndexDB) GetLatestSnapshotBlockLocation() (*chain_block.Location, error) {
	return nil, nil
}
