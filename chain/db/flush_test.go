package chain_db

import (
	"bytes"
	"github.com/vitelabs/go-vite/chain/flusher"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"gotest.tools/assert"
	"path"
	"sync"
	"testing"
)

func newStore() *Store {
	store, err := NewStore(path.Join(test_tools.DefaultDataDir(), "test_store", "store"), "store")
	if err != nil {
		panic(err)
	}

	return store
}

func TestRedoLog(t *testing.T) {
	store := newStore()
	var mu sync.RWMutex
	flusher, err := chain_flusher.NewFlusher([]chain_flusher.Storage{store}, &mu, path.Join(test_tools.DefaultDataDir(), "test_store"))
	// assert flusher
	assert.NilError(t, err)
	flusher.Flush()

}

func TestFlush(t *testing.T) {
	store := newStore()
	batch := store.NewBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))

	store.WriteDirectly(batch)

	// 1.prepare
	store.Prepare()
	c := newChecker(batch)
	// check flushing batch
	c.CheckBatch(store.flushingBatch)

	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)

	// 2.cancel prepare
	store.CancelPrepare()
	// check flushing batch
	assert.Assert(t, store.flushingBatch == nil || store.flushingBatch.Len() <= 0)

	// check snapshot batch
	c.CheckBatch(store.snapshotBatch)

	// 3.prepare
	store.Prepare()

	// check flushing batch
	c.CheckBatch(store.flushingBatch)

	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)

	// 4.redo log
	log, err := store.RedoLog()

	// assert nil error
	assert.NilError(t, err)

	// check flushing batch
	c.CheckBatch(store.flushingBatch)
	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)

	// check log
	assert.DeepEqual(t, log, store.flushingBatch.Dump())

	// check redo batch
	redoBatch := store.NewBatch()
	redoBatch.Load(log)

	c.CheckBatch(redoBatch)

	// 5.commit
	err = store.Commit()
	// assert nil error
	assert.NilError(t, err)
	// check flushing batch
	c.CheckBatch(store.flushingBatch)
	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)

	// check value
	v1, err := store.db.Get([]byte("key1"), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, v1, []byte("value1"))

	v2, err := store.db.Get([]byte("key2"), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, v2, []byte("value2"))

	v3, err := store.db.Get([]byte("key3"), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, v3, []byte("value3"))

	// 6.after commit
	store.AfterCommit()
	// check flushing batch
	assert.Assert(t, store.flushingBatch == nil || store.flushingBatch.Len() <= 0)
	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)
}

func TestRecover(t *testing.T) {
	store := newStore()
	batch := store.NewBatch()

	batch.Put([]byte("key1"), []byte("value1"))
	batch.Put([]byte("key2"), []byte("value2"))
	batch.Put([]byte("key3"), []byte("value3"))

	store.WriteDirectly(batch)

	// 1.prepare
	store.Prepare()
	c := newChecker(batch)
	// check flushing batch
	c.CheckBatch(store.flushingBatch)

	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)

	// 2.redo log
	log, err := store.RedoLog()

	// assert nil error
	assert.NilError(t, err)

	// check flushing batch
	c.CheckBatch(store.flushingBatch)
	// check snapshot batch
	assert.Assert(t, store.snapshotBatch == nil || store.snapshotBatch.Len() <= 0)

	// check log
	assert.DeepEqual(t, log, store.flushingBatch.Dump())

	// check redo batch
	redoBatch := store.NewBatch()
	redoBatch.Load(log)

	c.CheckBatch(redoBatch)

	// reset flushing batch and snapshot batch
	store.flushingBatch = nil
	store.snapshotBatch.Reset()

	// register AfterRecover
	callAfterRecover := false
	store.RegisterAfterRecover(func() {
		callAfterRecover = true
	})
	// 3.before recover
	store.BeforeRecover(log)

	// 4.recover
	store.PatchRedoLog(log)

	// 5. after recover
	store.AfterRecover()

	// check after recover
	assert.Equal(t, callAfterRecover, true)

	// check value
	v1, err := store.db.Get([]byte("key1"), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, v1, []byte("value1"))

	v2, err := store.db.Get([]byte("key2"), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, v2, []byte("value2"))

	v3, err := store.db.Get([]byte("key3"), nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, v3, []byte("value3"))

}

const (
	putFlag    = 1
	deleteFlag = 2
)

type checker struct {
	batch *leveldb.Batch
	data  [][3][]byte

	tmpData [][3][]byte
}

func newChecker(batch *leveldb.Batch) *checker {
	c := &checker{
		batch: batch,
		data:  make([][3][]byte, batch.Len()),
	}

	c.data = c.batchToData(batch)
	return c
}

func (c *checker) CheckBatch(b *leveldb.Batch) {
	data := c.batchToData(b)
	if len(data) != len(c.data) {
		panic("err")
	}

	for index, item := range data {
		cItem := c.data[index]
		if bytes.Compare(item[0], cItem[0]) != 0 ||
			bytes.Compare(item[1], cItem[1]) != 0 ||
			bytes.Compare(item[2], cItem[2]) != 0 {
			panic("err")
		}
	}

}

func (c *checker) batchToData(b *leveldb.Batch) [][3][]byte {
	b.Replay(c)
	return c.fetchData()
}

func (c *checker) fetchData() [][3][]byte {
	tmpData := c.tmpData
	c.tmpData = nil
	return tmpData
}

func (c *checker) Put(key []byte, value []byte) {
	c.tmpData = append(c.tmpData, [3][]byte{key, value, {putFlag}})
}

func (c *checker) Delete(key []byte) {
	c.tmpData = append(c.tmpData, [3][]byte{key, nil, {deleteFlag}})
}
