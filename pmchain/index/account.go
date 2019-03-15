package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"sync/atomic"
)

func (iDB *IndexDB) createAccount(blockHash *types.Hash, addr *types.Address) uint64 {
	newAccountId := atomic.AddUint64(&iDB.latestAccountId, 1)

	iDB.memDb.Put(blockHash, createAccountAddressKey(addr), SerializeAccountId(newAccountId))
	iDB.memDb.Put(blockHash, createAccountIdKey(newAccountId), addr.Bytes())
	return newAccountId

}
func (iDB *IndexDB) getAccountId(addr *types.Address) (uint64, error) {
	key := createAccountAddressKey(addr)
	value, ok := iDB.memDb.Get(key)
	if !ok {
		var err error
		value, err = iDB.store.Get(key)
		if err != nil {
			if err == leveldb.ErrNotFound {
				return 0, nil
			}
			return 0, err
		}
	}

	if len(value) <= 0 {
		return 0, nil
	}
	return DeserializeAccountId(value), nil
}

func (iDB *IndexDB) queryLatestAccountId() (uint64, error) {
	startKey := createAccountIdKey(1)
	endKey := createAccountIdKey(helper.MaxUint64)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	var latestAccountId uint64
	if iter.Last() {
		latestAccountId = FixedBytesToUint64(iter.Key()[1:])
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return 0, err
	}

	return latestAccountId, nil
}
