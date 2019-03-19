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
	key := chain_dbutils.CreateSnapshotBlockHashKey(hash)
	value, err := iDB.getValue(key)
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}
	height := chain_dbutils.DeserializeUint64(value)

	return iDB.GetSnapshotBlockLocation(height)
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

func (iDB *IndexDB) GetSnapshotBlockLocationList(blockHash *types.Hash, higher bool, count uint64) ([]*chain_block.Location, [2]uint64, error) {
	if count <= 0 {
		return nil, [2]uint64{}, nil
	}

	key := chain_dbutils.CreateSnapshotBlockHashKey(blockHash)
	value, err := iDB.getValue(key)
	if err != nil {
		return nil, [2]uint64{}, err
	}
	if len(value) <= 0 {
		return nil, [2]uint64{}, nil
	}
	height := chain_dbutils.DeserializeUint64(value)

	var startHeight, endHeight uint64
	if higher {
		startHeight = height
		endHeight = startHeight + count - 1
	} else {
		endHeight = height
		if endHeight <= count {
			startHeight = 1
		} else {
			startHeight = endHeight - count + 1
		}
	}

	return iDB.getSnapshotBlockLocations(startHeight, endHeight, higher)
}

func (iDB *IndexDB) GetRangeSnapshotBlockLocations(startHash *types.Hash, endHash *types.Hash) ([]*chain_block.Location, [2]uint64, error) {

	startKey := chain_dbutils.CreateSnapshotBlockHashKey(startHash)
	startValue, err := iDB.getValue(startKey)
	if err != nil {
		return nil, [2]uint64{}, err
	}
	if len(startValue) <= 0 {
		return nil, [2]uint64{}, nil
	}
	startHeight := chain_dbutils.DeserializeUint64(startValue)

	endKey := chain_dbutils.CreateSnapshotBlockHashKey(endHash)
	endValue, err := iDB.getValue(endKey)
	if err != nil {
		return nil, [2]uint64{}, err
	}
	if len(endValue) <= 0 {
		return nil, [2]uint64{}, nil
	}
	endHeight := chain_dbutils.DeserializeUint64(endValue)

	return iDB.getSnapshotBlockLocations(startHeight, endHeight, true)
}

func (iDB *IndexDB) getSnapshotBlockLocations(startHeight, endHeight uint64, higher bool) ([]*chain_block.Location, [2]uint64, error) {
	startKey := chain_dbutils.CreateSnapshotBlockHeightKey(startHeight)
	endKey := chain_dbutils.CreateSnapshotBlockHeightKey(endHeight + 1)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	locationList := make([]*chain_block.Location, 0, endHeight+1-startHeight)

	var minHeight, maxHeight uint64
	if higher {

		for iter.Next() {
			height := chain_dbutils.FixedBytesToUint64(iter.Key()[1:])
			if height < minHeight {
				minHeight = height
			}
			if height > maxHeight {
				maxHeight = height
			}

			locationList = append(locationList, chain_dbutils.DeserializeLocation(iter.Value()))
		}
	} else {
		iterOk := iter.Last()
		for iterOk {
			height := chain_dbutils.FixedBytesToUint64(iter.Key()[1:])
			if height < minHeight {
				minHeight = height
			}
			if height > maxHeight {
				maxHeight = height
			}

			locationList = append(locationList, chain_dbutils.DeserializeLocation(iter.Value()))
			iterOk = iter.Prev()
		}
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, [2]uint64{}, err
	}

	return locationList, [2]uint64{minHeight, maxHeight}, nil
}
