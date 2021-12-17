package chain_index

import (
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/hashicorp/golang-lru"

	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (iDB *IndexDB) newCache() error {
	var err error
	iDB.cache, err = bigcache.NewBigCache(bigcache.Config{
		HardMaxCacheSize: 128,
		Shards:           1024,
		LifeWindow:       time.Minute * 10,
	})
	if err != nil {
		return err
	}

	iDB.accountCache, err = lru.New(10 * 10000)
	if err != nil {
		return err
	}

	iDB.sendCreateBlockHashCache, err = lru.New(10000)
	if err != nil {
		return err
	}

	return nil
}

func (iDB *IndexDB) initCache(c Chain) error {
	var returnErr error
	c.IterateContracts(func(addr types.Address, meta *ledger.ContractMeta, err error) bool {
		if err != nil {
			returnErr = err
			return false
		}
		blockHash := meta.CreateBlockHash
		if !blockHash.IsZero() {
			snapshotHeight, err := iDB.GetConfirmHeightByHash(&blockHash)
			if err != nil {
				panic(fmt.Sprintf("indexDB initCache failed, Error: %s", err))
			}

			iDB.insertConfirmCache(blockHash, snapshotHeight)
		}
		return true
	})
	return returnErr
}

// with cache
func (iDB *IndexDB) getValue(key []byte) ([]byte, error) {
	value, err := iDB.cache.Get(string(key))
	if err != nil {
		if err == bigcache.ErrEntryNotFound {
			value, err = iDB.store.Get(key)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return value, nil
}
