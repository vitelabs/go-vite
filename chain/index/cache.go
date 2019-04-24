package chain_index

import (
	"github.com/allegro/bigcache"
	"github.com/hashicorp/golang-lru"
	"time"
)

func (iDB *IndexDB) newCache() error {
	var err error
	iDB.cache, err = bigcache.NewBigCache(bigcache.Config{
		HardMaxCacheSize: 256,
		Shards:           1024,
		LifeWindow:       time.Minute * 10,
	})
	if err != nil {
		return err
	}

	iDB.accountCache, err = lru.New(100 * 1000)
	if err != nil {
		return err
	}
	return nil
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
