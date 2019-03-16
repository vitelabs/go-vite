package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
)

func (iDB *IndexDB) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	key := chain_dbutils.CreateSnapshotBlockHashKey(hash)

	if ok := iDB.memDb.Has(key); ok {
		return ok, nil
	}

	return iDB.store.Has(key)
}

func (iDB *IndexDB) GetSnapshotBlockLocationByHash(hash *types.Hash) (*chain_block.Location, error) {
	chain_dbutils.CreateSnapshotBlockHashKey(hash)

	return nil, nil
}

func (iDB *IndexDB) GetSnapshotBlockLocation(height uint64) (*chain_block.Location, error) {
	key := chain_dbutils.CreateSnapshotBlockHeightKey(height)

	value, err := iDB.store.Get(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}

	return chain_dbutils.DeserializeLocation(value), nil
}

func (iDB *IndexDB) GetLatestSnapshotBlockLocation() (*chain_block.Location, error) {
	startKey := chain_dbutils.CreateSnapshotBlockHeightKey(1)
	endKey := chain_dbutils.CreateSnapshotBlockHeightKey(helper.MaxUint64)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	var location *chain_block.Location
	if iter.Last() {
		location = chain_dbutils.DeserializeLocation(iter.Value())
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return location, nil
}
