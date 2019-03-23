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

	height := chain_utils.FixedBytesToUint64(iter.Key()[9:])
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

// TODO
func (iDB *IndexDB) GetAccountBlockLocationList(hash *types.Hash, count uint64) ([]*chain_block.Location, uint64, [2]uint64, error) {

	return nil, 0, [2]uint64{}, nil
}
func (iDB *IndexDB) GetConfirmHeightByHash(blockHash *types.Hash) (uint64, error) {
	key := chain_utils.CreateConfirmHeightKey(blockHash)
	value, err := iDB.getValue(key)

	if err != nil {
		return 0, err
	}
	if len(value) <= 0 {
		return 0, nil
	}

	return chain_utils.FixedBytesToUint64(value), nil
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

func (iDB *IndexDB) getAccountIdHeight(blockHash *types.Hash) (uint64, uint64, error) {
	key := chain_utils.CreateAccountBlockHashKey(blockHash)
	value, err := iDB.getValue(key)
	if err != nil {
		return 0, 0, err
	}

	if len(value) <= 0 {
		return 0, 0, err
	}

	accountId, height := chain_utils.DeserializeAccountIdHeight(value)
	return accountId, height, nil
}
