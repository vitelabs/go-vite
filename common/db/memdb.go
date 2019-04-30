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

	mu sync.RWMutex
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
	atomic.AddUint64(&mDb.seq, 1)

	mDb.storage.Put(leveldb.MakeInternalKey(nil, key, mDb.seq, leveldb.KeyTypeVal), value)
}

func (mDb *MemDB) Delete(key []byte) {
	atomic.AddUint64(&mDb.seq, 1)

	mDb.storage.Put(leveldb.MakeInternalKey(nil, key, mDb.seq, leveldb.KeyTypeDel), nil)
}
