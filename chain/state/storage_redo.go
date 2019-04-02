package chain_state

import (
	"github.com/boltdb/bolt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"path"
	"sync"
)

type StorageRedo struct {
	store          *bolt.DB
	logMap         map[types.Hash][]byte
	snapshotHeight uint64
	retainHeight   uint64

	id types.Hash
	mu sync.RWMutex

	flushingBatch *leveldb.Batch
}

func NewStorageRedo(chainDir string) (*StorageRedo, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("state_db_kv_redo")))

	store, err := bolt.Open(path.Join(chainDir, "kv_redo"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return &StorageRedo{
		store:        store,
		logMap:       make(map[types.Hash][]byte),
		retainHeight: 600,
		id:           id,

		flushingBatch: new(leveldb.Batch),
	}, nil
}

func (redo *StorageRedo) SetSnapshot(snapshotHeight uint64, redoLog map[types.Hash][]byte) {
	redo.logMap = redoLog
	if redo.logMap == nil {
		redo.logMap = make(map[types.Hash][]byte)
	}
	redo.snapshotHeight = snapshotHeight
}

func (redo *StorageRedo) AddLog(blockHash types.Hash, log []byte) {
	redo.mu.Lock()
	defer redo.mu.Unlock()

	redo.logMap[blockHash] = log
}

func (redo *StorageRedo) Id() types.Hash {
	return redo.id
}

func (redo *StorageRedo) Prepare() {
	redo.flushingBatch.Reset()
	for blockHash, kvLog := range redo.logMap {
		redo.flushingBatch.Put(blockHash.Bytes(), kvLog)
	}
}

func (redo *StorageRedo) CancelPrepare() {
	redo.flushingBatch.Reset()
}

func (redo *StorageRedo) RedoLog() ([]byte, error) {
	if redo.flushingBatch.Len() <= 0 {
		return nil, nil
	}

	redoLog := append(chain_utils.Uint64ToBytes(redo.snapshotHeight), redo.flushingBatch.Dump()...)

	return redoLog, nil
}

func (redo *StorageRedo) Commit() error {
	if redo.flushingBatch.Len() <= 0 {
		return nil
	}

	defer func() {
		redo.flushingBatch.Reset()
	}()

	return redo.flush(redo.snapshotHeight, redo.flushingBatch)
}

func (redo *StorageRedo) PatchRedoLog(redoLog []byte) error {
	batch := new(leveldb.Batch)
	if err := batch.Load(redoLog[8:]); err != nil {
		return err
	}
	if batch.Len() <= 0 {
		return nil
	}
	return redo.flush(chain_utils.BytesToUint64(redoLog[:8]), batch)
}

func (redo *StorageRedo) flush(snapshotHeight uint64, batch *leveldb.Batch) error {
	tx, err := redo.store.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bu, err := tx.CreateBucketIfNotExists(chain_utils.Uint64ToBytes(snapshotHeight))
	if err != nil {
		return err
	}

	// add
	redo.flushingBatch.Replay(NewBatchFlush(bu))

	// delete
	if snapshotHeight > redo.retainHeight {
		// ignore error
		tx.DeleteBucket(chain_utils.Uint64ToBytes(snapshotHeight - redo.retainHeight))
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

type BatchFlush struct {
	bu *bolt.Bucket
}

func NewBatchFlush(bu *bolt.Bucket) *BatchFlush {
	return &BatchFlush{
		bu: bu,
	}
}

func (flush *BatchFlush) Put(key []byte, value []byte) {
	if err := flush.bu.Put(key, value); err != nil {
		panic(err)
	}
}

func (flush *BatchFlush) Delete(key []byte) {
	if err := flush.bu.Delete(key); err != nil {
		panic(err)
	}
}
