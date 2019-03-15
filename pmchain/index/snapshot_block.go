package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (iDB *IndexDB) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	key := createSnapshotBlockHashKey(hash)

	if ok := iDB.memDb.Has(key); ok {
		return ok, nil
	}

	return iDB.store.Has(key)
}

func (iDB *IndexDB) GetSnapshotBlockLocationByHash(hash *types.Hash) (*chain_block.Location, error) {
	createSnapshotBlockHashKey(hash)

	return nil, nil
}

func (iDB *IndexDB) GetSnapshotBlockLocation(height uint64) (*chain_block.Location, error) {
	key := createSnapshotBlockHeightKey(height)

	value, err := iDB.store.Get(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}

	return DeserializeLocation(value), nil
}

func (iDB *IndexDB) GetLatestSnapshotBlockLocation() (*chain_block.Location, error) {
	startKey := createSnapshotBlockHeightKey(1)
	endKey := createSnapshotBlockHeightKey(helper.MaxUint64)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	var location *chain_block.Location
	if iter.Last() {
		location = DeserializeLocation(iter.Value())
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return location, nil
}
