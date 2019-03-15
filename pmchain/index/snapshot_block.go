package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (iDB *IndexDB) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	key := make([]byte, 0, 1+types.HashSize)
	key = append(append(append(key, SnapshotBlockHashKeyPrefix), hash.Bytes()...))

	if ok := iDB.memDb.Has(key); ok {
		return ok, nil
	}

	return iDB.store.Has(key)
}

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
