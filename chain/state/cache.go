package chain_state

import (
	"github.com/allegro/bigcache"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	snapshotValuePrefix = "a"
	contractAddrPrefix  = "b"
)

func (sDB *StateDB) newCache() error {
	var err error

	sDB.cache, err = bigcache.NewBigCache(bigcache.Config{
		Shards: 1024,
	})
	if err != nil {
		return err
	}
	return nil
}

// with cache
func (sDB *StateDB) initCache() error {
	latestDB := sDB.chain.GetLatestSnapshotBlock()
	for _, contractAddr := range types.BuiltinContractAddrList {
		addrStr := string(contractAddr.Bytes())

		if meta := ledger.GetBuiltinContractMeta(contractAddr); meta != nil {
			sDB.cache.Set(contractAddrPrefix+addrStr, meta.Serialize())
		}

		db, err := sDB.NewStorageDatabase(latestDB.Hash, contractAddr)
		if err != nil {
			return err
		}
		iter, err := db.NewStorageIterator(nil)
		if err != nil {
			return err
		}

		for iter.Next() {
			key := iter.Key()
			value := iter.Value()

			sDB.cache.Set(snapshotValuePrefix+addrStr+string(key), value)
		}

		iterErr := iter.Error()
		iter.Release()
		if iterErr != nil {
			return iterErr
		}
	}

	if err := sDB.initContractMetaCache(); err != nil {
		return err
	}

	return nil
}

func (sDB *StateDB) initContractMetaCache() error {
	iter := sDB.store.NewIterator(util.BytesPrefix([]byte{chain_utils.ContractMetaKeyPrefix}))
	defer iter.Release()

	for iter.Next() {
		addrKey := iter.Key()
		addrStr := string(addrKey[1:])

		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())

		sDB.cache.Set(contractAddrPrefix+addrStr, value)
	}
	if err := iter.Error(); err != nil {
		return err
	}
	return nil
}
