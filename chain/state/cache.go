package chain_state

import (
	"github.com/patrickmn/go-cache"

	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
)

const (
	snapshotValuePrefix = "a"
	contractAddrPrefix  = "b"
	balancePrefix       = "c"
)

func (sDB *StateDB) newCache() error {
	var err error

	sDB.cache = cache.New(cache.NoExpiration, cache.NoExpiration)

	if err != nil {
		return err
	}
	return nil
}

// with cache
func (sDB *StateDB) initCache() error {
	if err := sDB.initSnapshotValueCache(); err != nil {
		return err
	}

	if err := sDB.initContractMetaCache(); err != nil {
		return err
	}

	return nil
}

func (sDB *StateDB) disableCache() {
	sDB.useCache = false
}

func (sDB *StateDB) enableCache() {
	sDB.useCache = true
}

func (sDB *StateDB) initSnapshotValueCache() error {
	for _, contractAddr := range types.BuiltinContractAddrList {

		iter := sDB.NewStorageIterator(&contractAddr, nil)
		for iter.Next() {
			//fmt.Printf("init set snapshot value cache: %d, %d\n", []byte(snapshotValuePrefix+addrStr+string(key)), sDB.copyValue(value))
			sDB.cache.Set(snapshotValuePrefix+string(append(contractAddr.Bytes(), iter.Key()...)), sDB.copyValue(iter.Value()), cache.NoExpiration)
		}

		returnErr := iter.Error()
		iter.Release()

		if returnErr != nil {
			return returnErr
		}

	}

	return nil
}

func (sDB *StateDB) initContractMetaCache() error {
	iter := sDB.store.NewIterator(util.BytesPrefix([]byte{chain_utils.ContractMetaKeyPrefix}))
	defer iter.Release()

	for iter.Next() {
		sDB.cache.Set(contractAddrPrefix+string(iter.Key()), sDB.copyValue(iter.Value()), cache.NoExpiration)
	}
	if err := iter.Error(); err != nil {
		return err
	}
	return nil
}

// with cache
func (sDB *StateDB) getValue(key []byte, cachePrefix string) ([]byte, error) {
	ok := false
	var value interface{}
	if sDB.useCache {
		value, ok = sDB.cache.Get(cachePrefix + string(key))
	}
	if !ok {
		var err error
		value, err = sDB.store.Get(key)
		if err != nil {
			return nil, err
		}
	}
	return value.([]byte), nil
}

func (sDB *StateDB) getValueInCache(key []byte, cachePrefix string) ([]byte, error) {
	if !sDB.useCache {
		return sDB.getValue(key, cachePrefix)
	}
	value, ok := sDB.cache.Get(cachePrefix + string(key))
	if !ok {
		return nil, nil
	}
	return value.([]byte), nil
}

func (sDB *StateDB) parseStorageKey(key []byte) []byte {
	realKey := key[1+types.AddressSize : 1+types.AddressSize+types.HashSize+1]
	return realKey[:realKey[len(realKey)-1]]
}

func (sDB *StateDB) copyValue(value []byte) []byte {
	newValue := make([]byte, len(value))
	copy(newValue, value)
	return newValue
}
