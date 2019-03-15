package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync/atomic"
)

func (iDB *IndexDB) createAccount(blockHash *types.Hash, addr *types.Address) uint64 {
	newAccountId := atomic.AddUint64(&iDB.latestAccountId, 1)

	iDB.memDb.Put(blockHash, getAccountAddressKey(addr), SerializeAccountId(newAccountId))
	return newAccountId

}
func (iDB *IndexDB) getAccountId(addr *types.Address) (uint64, error) {
	key := getAccountAddressKey(addr)
	value, ok := iDB.memDb.Get(key)
	if !ok {
		var err error
		value, err = iDB.store.Get(key)
		if err != nil {
			return 0, err
		}
	}

	if len(value) <= 0 {
		return 0, nil
	}
	return DeserializeAccountId(value), nil
}

func (iDB *IndexDB) queryLatestAccountId() (uint64, error) {

	return 0, nil
}
