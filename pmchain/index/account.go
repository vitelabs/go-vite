package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
	"sync/atomic"
)

func (iDB *IndexDB) createAccount(blockHash *types.Hash, addr *types.Address) uint64 {
	newAccountId := atomic.AddUint64(&iDB.latestAccountId, 1)

	iDB.memDb.Put(blockHash, chain_dbutils.CreateAccountAddressKey(addr), chain_dbutils.SerializeAccountId(newAccountId))
	iDB.memDb.Put(blockHash, chain_dbutils.CreateAccountIdKey(newAccountId), addr.Bytes())
	return newAccountId

}
func (iDB *IndexDB) getAccountId(addr *types.Address) (uint64, error) {
	key := chain_dbutils.CreateAccountAddressKey(addr)
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
	return chain_dbutils.DeserializeAccountId(value), nil
}

func (iDB *IndexDB) queryLatestAccountId() (uint64, error) {
	startKey := chain_dbutils.CreateAccountIdKey(1)
	endKey := chain_dbutils.CreateAccountIdKey(helper.MaxUint64)

	iter := iDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
	defer iter.Release()

	var latestAccountId uint64
	if iter.Last() {
		latestAccountId = chain_dbutils.FixedBytesToUint64(iter.Key()[1:])
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return 0, err
	}

	return latestAccountId, nil
}
