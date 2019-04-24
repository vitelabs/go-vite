package chain_db

import (
	"github.com/patrickmn/go-cache"
	"github.com/vitelabs/go-vite/interfaces"
)

//const (
//	insertFlag = byte(1)
//	deleteFlag = byte(2)
//)

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
	//allDeletedStore := db.deletedStore.Items()

	//sortedItems := make(SortedItems, 0, len(allItems)+len(allDeletedStore))
	//
	//for keyStr, item := range allItems {
	//	sortedItems = append(sortedItems, [3][]byte{[]byte(keyStr), item.Object.([]byte), {insertFlag}})
	//}
	//
	//for keyStr := range allDeletedStore {
	//	sortedItems = append(sortedItems, [3][]byte{[]byte(keyStr), nil, {deleteFlag}})
	//	//batch.Delete([]byte(key))
	//}

	//sort.Sort(sortedItems)

	//for _, kv := range sortedItems {
	//	switch kv[2][0] {
	//	case insertFlag:
	//		batch.Put(kv[0], kv[1])
	//	case deleteFlag:
	//		batch.Delete(kv[0])
	//	}
	//}

	for keyStr, value := range allItems {
		batch.Put([]byte(keyStr), value.Object.([]byte))
	}

	allDeletedStore := db.deletedStore.Items()
	//fmt.Println(len(allDeletedStore))

	for key := range allDeletedStore {
		batch.Delete([]byte(key))
	}

}

func (db *SnapshotMemDB) Reset() {
	db.store.Flush()
	db.deletedStore.Flush()
}

//type SortedItems [][3][]byte
//
//func (items SortedItems) Len() int {
//	return len(items)
//}
//
//func (items SortedItems) Swap(i, j int) {
//	items[i], items[j] = items[j], items[i]
//}
//
//func (items SortedItems) Less(i, j int) bool {
//	return bytes.Compare(items[i][0], items[j][0]) < 0
//}
