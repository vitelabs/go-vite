package chain_db

import (
	"github.com/patrickmn/go-cache"
	"github.com/vitelabs/go-vite/interfaces"
)

type SnapshotMemDB struct {
	store        *cache.Cache
	deletedStore *cache.Cache
}

func NewSnapshotMemDB() *SnapshotMemDB {

	return &SnapshotMemDB{
		store:        cache.New(cache.NoExpiration, cache.NoExpiration),
		deletedStore: cache.New(cache.NoExpiration, cache.NoExpiration),
	}
}

func (db *SnapshotMemDB) Put(key, value []byte) {
	db.store.Set(string(key), value, cache.NoExpiration)
	db.deletedStore.Delete(string(key))
}

func (db *SnapshotMemDB) Delete(key []byte) {
	db.store.Delete(string(key))
	db.deletedStore.Set(string(key), nil, cache.NoExpiration)
}

func (db *SnapshotMemDB) Flush(batch interfaces.Batch) {
	allItems := db.store.Items()
	for key, value := range allItems {
		batch.Put([]byte(key), value.Object.([]byte))
	}

	allDeletedStore := db.deletedStore.Items()
	for key := range allDeletedStore {
		batch.Delete([]byte(key))
	}
}

func (db *SnapshotMemDB) Reset() {
	db.store.Flush()
	db.deletedStore.Flush()
}
