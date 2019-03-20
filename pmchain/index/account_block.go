package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/utils"
)

func (iDB *IndexDB) IsAccountBlockExisted(hash *types.Hash) (bool, error) {
	key := chain_utils.CreateAccountBlockHashKey(hash)

	return iDB.hasValue(key)
}

// latest account block in disk
func (iDB *IndexDB) GetLatestAccountBlock(addr *types.Address) (uint64, *chain_block.Location, error) {
	accountId, err := iDB.GetAccountId(addr)
	if err != nil {
		return 0, nil, err
	}

	startKey := chain_utils.CreateAccountBlockHeightKey(accountId, 1)
	endKey := chain_utils.CreateAccountBlockHeightKey(accountId, helper.MaxUint64)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	if !iter.Last() {
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return 0, nil, nil
		}
		return 0, nil, err
	}

	height := chain_utils.FixedBytesToUint64(iter.Key()[9:])
	location := chain_utils.DeserializeLocation(iter.Value())
	return height, location, nil
}

func (iDB *IndexDB) GetAccountBlockLocation(addr *types.Address, height uint64) (*chain_block.Location, error) {
	accountId, err := iDB.GetAccountId(addr)
	if err != nil {
		return nil, err
	}
	key := chain_utils.CreateAccountBlockHeightKey(accountId, height)
	value, err := iDB.getValue(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return chain_utils.DeserializeLocation(value), nil
}

// TODO
func (iDB *IndexDB) GetAccountBlockLocationList(hash *types.Hash, count uint64) ([]*chain_block.Location, uint64, [2]uint64, error) {

	return nil, 0, [2]uint64{}, nil
}
func (iDB *IndexDB) GetConfirmHeightByHash(blockHash *types.Hash) (uint64, error) {
	accountId, height, err := iDB.getAccountIdHeight(blockHash)
	if err != nil {
		return 0, err
	}

	if accountId <= 0 {
		return 0, nil
	}
	key := chain_utils.CreateConfirmHeightKey(accountId, height)
	value, err := iDB.getValue(key)

	if err != nil {
		return 0, err
	}
	if len(value) <= 0 {
		return 0, nil
	}

	return chain_utils.FixedBytesToUint64(value), nil
}

func (iDB *IndexDB) GetReceivedBySend(sendBlockHash *types.Hash) (uint64, uint64, error) {
	accountId, height, err := iDB.getAccountIdHeight(sendBlockHash)
	if err != nil {
		return 0, 0, err
	}

	if accountId <= 0 {
		return 0, 0, nil
	}
	key := chain_utils.CreateReceiveHeightKey(accountId, height)
	value, err := iDB.getValue(key)

	if err != nil {
		return 0, 0, err
	}
	if len(value) <= 0 {
		return 0, 0, nil
	}

	receiveAccountId, receiveHeight := chain_utils.DeserializeAccountIdHeight(value)
	return receiveAccountId, receiveHeight, nil
}

func (iDB *IndexDB) IsReceived(sendBlockHash *types.Hash) (bool, error) {
	accountId, height, err := iDB.getAccountIdHeight(sendBlockHash)
	if err != nil {
		return false, err
	}

	if accountId <= 0 {
		return false, nil
	}

	key := chain_utils.CreateReceiveHeightKey(accountId, height)
	return iDB.hasValue(key)
}
func (iDB *IndexDB) GetVmLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	key := chain_utils.CreateVmLogListKey(logHash)
	value, err := iDB.getValue(key)
	if err != nil {
		return nil, err
	}

	if len(value) <= 0 {
		return nil, nil
	}
	return ledger.VmLogListDeserialize(value)
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
