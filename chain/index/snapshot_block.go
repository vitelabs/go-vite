package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
)

func (iDB *IndexDB) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	return iDB.store.Has(chain_utils.CreateSnapshotBlockHashKey(hash))
}

func (iDB *IndexDB) GetSnapshotBlockHeight(hash *types.Hash) (uint64, error) {
	key := chain_utils.CreateSnapshotBlockHashKey(hash)
	value, err := iDB.store.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	if len(value) <= 0 {
		return 0, nil
	}
	return chain_utils.BytesToUint64(value), nil

}

func (iDB *IndexDB) GetSnapshotBlockLocationByHash(hash *types.Hash) (*chain_file_manager.Location, error) {
	key := chain_utils.CreateSnapshotBlockHashKey(hash)
	value, err := iDB.store.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}

	return iDB.GetSnapshotBlockLocation(chain_utils.BytesToUint64(value))
}

func (iDB *IndexDB) GetSnapshotBlockLocation(height uint64) (*chain_file_manager.Location, error) {
	key := chain_utils.CreateSnapshotBlockHeightKey(height)

	value, err := iDB.store.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}

	return chain_utils.DeserializeLocation(value[types.HashSize:]), nil
}

// only in disk
func (iDB *IndexDB) GetLatestSnapshotBlockLocation() (*chain_file_manager.Location, error) {
	startKey := chain_utils.CreateSnapshotBlockHeightKey(1)
	endKey := chain_utils.CreateSnapshotBlockHeightKey(helper.MaxUint64)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	var location *chain_file_manager.Location
	if iter.Last() {
		location = chain_utils.DeserializeLocation(iter.Value()[types.HashSize:])
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return location, nil
}

func (iDB *IndexDB) GetSnapshotBlockLocationList(blockHash *types.Hash, higher bool, count uint64) ([]*chain_file_manager.Location, [2]uint64, error) {
	if count <= 0 {
		return nil, [2]uint64{}, nil
	}

	key := chain_utils.CreateSnapshotBlockHashKey(blockHash)
	value, err := iDB.store.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, [2]uint64{}, nil
		}
		return nil, [2]uint64{}, err

	}
	if len(value) <= 0 {
		return nil, [2]uint64{}, err

	}
	height := chain_utils.BytesToUint64(value)

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

func (iDB *IndexDB) GetRangeSnapshotBlockLocations(startHash *types.Hash, endHash *types.Hash) ([]*chain_file_manager.Location, [2]uint64, error) {

	startKey := chain_utils.CreateSnapshotBlockHashKey(startHash)
	startValue, err := iDB.store.Get(startKey)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, [2]uint64{}, nil
		}
		return nil, [2]uint64{}, err
	}
	if len(startValue) <= 0 {
		return nil, [2]uint64{}, nil
	}
	startHeight := chain_utils.BytesToUint64(startValue)

	endKey := chain_utils.CreateSnapshotBlockHashKey(endHash)
	endValue, err := iDB.store.Get(endKey)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, [2]uint64{}, nil
		}
		return nil, [2]uint64{}, err
	}
	if len(endValue) <= 0 {
		return nil, [2]uint64{}, nil
	}
	endHeight := chain_utils.BytesToUint64(endValue)

	return iDB.getSnapshotBlockLocations(startHeight, endHeight, true)
}

func (iDB *IndexDB) getSnapshotBlockLocations(startHeight, endHeight uint64, higher bool) ([]*chain_file_manager.Location, [2]uint64, error) {
	startKey := chain_utils.CreateSnapshotBlockHeightKey(startHeight)
	endKey := chain_utils.CreateSnapshotBlockHeightKey(endHeight + 1)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	locationList := make([]*chain_file_manager.Location, 0, endHeight+1-startHeight)

	var minHeight, maxHeight uint64
	if higher {

		for iter.Next() {
			height := chain_utils.BytesToUint64(iter.Key()[1:])
			if height < minHeight {
				minHeight = height
			}
			if height > maxHeight {
				maxHeight = height
			}

			locationList = append(locationList, chain_utils.DeserializeLocation(iter.Value()[types.HashSize:]))
		}
	} else {
		iterOk := iter.Last()
		for iterOk {
			height := chain_utils.BytesToUint64(iter.Key()[1:])
			if height < minHeight {
				minHeight = height
			}
			if height > maxHeight {
				maxHeight = height
			}

			locationList = append(locationList, chain_utils.DeserializeLocation(iter.Value()[types.HashSize:]))
			iterOk = iter.Prev()
		}
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, [2]uint64{}, err
	}

	return locationList, [2]uint64{minHeight, maxHeight}, nil
}
