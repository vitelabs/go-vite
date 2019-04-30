package chain_index

import (
	"github.com/allegro/bigcache"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
)

func (iDB *IndexDB) IsSnapshotBlockExisted(hash *types.Hash) (bool, error) {
	return iDB.store.Has(chain_utils.CreateSnapshotBlockHashKey(hash))
}

func (iDB *IndexDB) GetSnapshotBlockHeight(hash *types.Hash) (uint64, error) {
	value, err := iDB.getValue(chain_utils.CreateSnapshotBlockHashKey(hash))
	if err != nil {
		return 0, err
	}
	if len(value) <= 0 {
		return 0, nil
	}
	return chain_utils.BytesToUint64(value), nil
}

func (iDB *IndexDB) GetSnapshotBlockLocationByHash(hash *types.Hash) (*chain_file_manager.Location, error) {
	value, err := iDB.getValue(chain_utils.CreateSnapshotBlockHashKey(hash))
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

	value, err := iDB.getValue(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}

	return chain_utils.DeserializeLocation(value[types.HashSize:]), nil
}

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

	value, err := iDB.getValue(chain_utils.CreateSnapshotBlockHashKey(blockHash))
	if err != nil {
		return nil, [2]uint64{}, err
	}

	if len(value) <= 0 {
		return nil, [2]uint64{}, err

	}
	height := chain_utils.BytesToUint64(value)

	return iDB.GetSnapshotBlockLocationListByHeight(height, higher, count)
}

func (iDB *IndexDB) GetSnapshotBlockLocationListByHeight(height uint64, higher bool, count uint64) ([]*chain_file_manager.Location, [2]uint64, error) {
	if count <= 0 {
		return nil, [2]uint64{}, nil
	}

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

	return iDB.getSnapshotBlockLocations(startHeight, endHeight)
}

func (iDB *IndexDB) GetRangeSnapshotBlockLocations(startHash *types.Hash, endHash *types.Hash) ([]*chain_file_manager.Location, [2]uint64, error) {

	startValue, err := iDB.getValue(chain_utils.CreateSnapshotBlockHashKey(startHash))
	if err != nil {
		return nil, [2]uint64{}, err
	}
	if len(startValue) <= 0 {
		return nil, [2]uint64{}, nil
	}
	startHeight := chain_utils.BytesToUint64(startValue)

	endValue, err := iDB.getValue(chain_utils.CreateSnapshotBlockHashKey(endHash))
	if err != nil {
		return nil, [2]uint64{}, err
	}
	if len(endValue) <= 0 {
		return nil, [2]uint64{}, nil
	}
	endHeight := chain_utils.BytesToUint64(endValue)

	return iDB.getSnapshotBlockLocations(startHeight, endHeight)
}

func (iDB *IndexDB) getSnapshotBlockLocations(startHeight, endHeight uint64) ([]*chain_file_manager.Location, [2]uint64, error) {
	locationList, heightRange, err := iDB.getSnapshotBlockLocationsByCache(endHeight, startHeight)

	if err != nil {
		return nil, [2]uint64{}, err
	}

	maxHeight := heightRange[0]
	minHeight := heightRange[1]

	if minHeight <= endHeight {
		endHeight = minHeight - 1
	}

	if endHeight >= startHeight {
		startKey := chain_utils.CreateSnapshotBlockHeightKey(startHeight)
		endKey := chain_utils.CreateSnapshotBlockHeightKey(endHeight + 1)

		iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
		defer iter.Release()

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

		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return nil, [2]uint64{}, err
		}
	}

	return locationList, [2]uint64{maxHeight, minHeight}, nil
}

func (iDB *IndexDB) getSnapshotBlockLocationsByCache(endHeight, startHeight uint64) ([]*chain_file_manager.Location, [2]uint64, error) {
	h := endHeight

	locationList := make([]*chain_file_manager.Location, 0, endHeight+1-startHeight)
	minHeight := endHeight + 1
	maxHeight := startHeight - 1

	for ; h >= startHeight; h-- {
		value, err := iDB.cache.Get(string(chain_utils.CreateSnapshotBlockHeightKey(startHeight)))
		if err != nil {
			if err == bigcache.ErrEntryNotFound {
				break
			}
			return nil, [2]uint64{0, 0}, err
		}
		locationList = append(locationList, chain_utils.DeserializeLocation(value[types.HashSize:]))

		if h < minHeight {
			minHeight = h
		}
		if h > maxHeight {
			maxHeight = h
		}
	}
	//iDB.cache.Get()
	return locationList, [2]uint64{maxHeight, minHeight}, nil
}
