package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
)

func (iDB *IndexDB) IsAccountBlockExisted(hash *types.Hash) (bool, error) {
	return iDB.hasValue(chain_utils.CreateAccountBlockHashKey(hash))
}

func (iDB *IndexDB) GetLatestAccountBlock(addr *types.Address) (uint64, *chain_block.Location, error) {

	startKey := chain_utils.CreateAccountBlockHeightKey(addr, 1)
	endKey := chain_utils.CreateAccountBlockHeightKey(addr, helper.MaxUint64)

	iter := iDB.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	if !iter.Last() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return 0, nil, err
		}
		return 0, nil, nil
	}

	height := chain_utils.BytesToUint64(iter.Key()[9:])
	location := chain_utils.DeserializeLocation(iter.Value()[types.HashSize:])
	return height, location, nil
}

func (iDB *IndexDB) GetAccountBlockLocation(addr *types.Address, height uint64) (*chain_block.Location, error) {
	key := chain_utils.CreateAccountBlockHeightKey(addr, height)
	value, err := iDB.getValue(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return chain_utils.DeserializeLocation(value[types.HashSize:]), nil
}

func (iDB *IndexDB) GetAccountBlockLocationList(hash *types.Hash, count uint64) (*types.Address, []*chain_block.Location, [2]uint64, error) {
	if count <= 0 {
		return nil, nil, [2]uint64{}, nil
	}

	addr, height, err := iDB.GetAddrHeightByHash(hash)
	if err != nil {
		return nil, nil, [2]uint64{}, err
	}
	if addr == nil {
		return nil, nil, [2]uint64{}, nil
	}

	startHeight := uint64(1)

	endHeight := height
	if endHeight > count {
		startHeight = endHeight - count + 1
	}

	startKey := chain_utils.CreateAccountBlockHeightKey(addr, startHeight)
	endKey := chain_utils.CreateAccountBlockHeightKey(addr, endHeight+1)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	locationList := make([]*chain_block.Location, 0, endHeight+1-startHeight)

	var minHeight, maxHeight uint64

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
		return nil, nil, [2]uint64{}, err
	}

	return addr, locationList, [2]uint64{minHeight, maxHeight}, nil
}

func (iDB *IndexDB) GetConfirmHeightByHash(blockHash *types.Hash) (uint64, error) {
	addr, height, err := iDB.GetAddrHeightByHash(blockHash)
	if err != nil {
		return 0, err
	}
	if addr == nil {
		return 0, nil
	}

	startKey := chain_utils.CreateConfirmHeightKey(addr, height)
	endKey := chain_utils.CreateConfirmHeightKey(addr, helper.MaxUint64)

	iter := iDB.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	for iter.Next() {
		value := iter.Value()
		return chain_utils.BytesToUint64(value), nil
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return 0, err
	}

	return 0, nil
}

func (iDB *IndexDB) GetReceivedBySend(sendBlockHash *types.Hash) (*types.Hash, error) {

	value, err := iDB.getValue(chain_utils.CreateReceiveKey(sendBlockHash))
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}

	hash, err := types.BytesToHash(value)
	if err != nil {
		return nil, err
	}
	return &hash, nil
}

func (iDB *IndexDB) IsReceived(sendBlockHash *types.Hash) (bool, error) {
	return iDB.hasValue(chain_utils.CreateReceiveKey(sendBlockHash))
}

func (iDB *IndexDB) GetAddrHeightByHash(blockHash *types.Hash) (*types.Address, uint64, error) {

	key := chain_utils.CreateAccountBlockHashKey(blockHash)
	value, err := iDB.store.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, 0, nil
		}
		return nil, 0, err

	}
	if len(value) <= 0 {
		return nil, 0, nil

	}

	addr, err := types.BytesToAddress(value[:types.AddressSize])
	if err != nil {
		return nil, 0, err
	}
	height := chain_utils.BytesToUint64(value[types.AddressSize:])
	return &addr, height, nil
}
