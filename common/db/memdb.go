package db

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/comparer"
	"github.com/vitelabs/go-vite/common/db/xleveldb/memdb"
	"sync"
	"sync/atomic"
)

type MemDB struct {
	storage *memdb.DB

	seq uint64

	copyMu sync.RWMutex
}

func NewMemDB() *MemDB {
	return &MemDB{
		storage: newStorage(),
		seq:     leveldb.KeyMaxSeq - 10000*10000,
	}
}

func newStorage() *memdb.DB {
	return memdb.New(leveldb.NewIComparer(comparer.DefaultComparer), 0)
}

func (mDb *MemDB) GetDb() *memdb.DB {
	return mDb.storage
}

func (mDb *MemDB) GetSeq() uint64 {
	return atomic.LoadUint64(&mDb.seq)
}

func (mDb *MemDB) Put(key, value []byte) {
	//mDb.copyMu.RLock()
	//defer mDb.copyMu.RUnlock()

	atomic.AddUint64(&mDb.seq, 1)

	mDb.storage.Put(leveldb.MakeInternalKey(nil, key, mDb.seq, leveldb.KeyTypeVal), value)
}

func (mDb *MemDB) Delete(key []byte) {
	//mDb.copyMu.RLock()
	//defer mDb.copyMu.RUnlock()

	atomic.AddUint64(&mDb.seq, 1)
	mDb.storage.Put(leveldb.MakeInternalKey(nil, key, mDb.seq, leveldb.KeyTypeDel), nil)
}

func (mDb *MemDB) Len() int {
	return mDb.storage.Len()
}

func (mDb *MemDB) Size() int {
	return mDb.storage.Size()
}

//const CopyLimitDefault = uint64(9000 * 10000)
//
//var CopyLimit = CopyLimitDefault
//
//func (mDb *MemDB) Copy() *MemDB {
//	mDb.copyMu.Lock()
//	defer mDb.copyMu.Unlock()
//
//	newMemDB := NewMemDB()
//
//	seq := atomic.LoadUint64(&mDb.seq)
//	if seq >= CopyLimit {
//		leveldb.
//		iter := mDb.GetDb().NewIterator(nil)
//		defer iter.Release()
//
//		for iter.Next() {
//			newMemDB.Put(iter.Key(), iter.Value())
//		}
//		if err := iter.Error(); err != nil {
//			panic(fmt.Sprintf("MemDB copy failed, Error: %s", err.Error()))
//		}
//
//		// copy storage
//
//		return nil
//	} else {
//		// copy seq
//		newMemDB.seq = seq
//		// copy storage
//		newMemDB.storage = mDb.storage.Copy()
//	}
//
//	return newMemDB
//
//}
