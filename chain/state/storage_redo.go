package chain_state

import (
	"encoding/binary"
	"path"
	"sync"

	"fmt"
	"github.com/boltdb/bolt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

const (
	optFlush    = 1
	optRollback = 2
)

type StorageRedo struct {
	store          *bolt.DB
	logMap         map[types.Hash][]byte
	snapshotHeight uint64
	retainHeight   uint64
	hasRedo        bool

	id types.Hash
	mu sync.RWMutex

	flushingBatch   *leveldb.Batch
	rollbackHeights []uint64
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

func (redo *StorageRedo) SetSnapshot(snapshotHeight uint64, redoLog map[types.Hash][]byte, hasRedo bool) {
	redo.logMap = redoLog

	if redo.logMap == nil {
		redo.logMap = make(map[types.Hash][]byte)
	}
	redo.snapshotHeight = snapshotHeight
	redo.hasRedo = hasRedo
}

func (redo *StorageRedo) QueryLog(snapshotHeight uint64) (map[types.Hash][]byte, bool, error) {
	if snapshotHeight == redo.snapshotHeight {
		return redo.logMap, true, nil
	}
	logMap := make(map[types.Hash][]byte)

	hasRedo := false
	err := redo.store.View(func(tx *bolt.Tx) error {
		bu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight))
		if bu == nil {
			return nil
		}
		hasRedo = true

		c := bu.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			hash, err := types.BytesToHash(k)
			if err != nil {
				return err
			}
			logMap[hash] = v
		}

		return nil
	})

	return logMap, hasRedo, err
}

func (redo *StorageRedo) HasRedo() bool {
	return redo.hasRedo
}

func (redo *StorageRedo) AddLog(blockHash types.Hash, log []byte) {
	redo.mu.Lock()
	defer redo.mu.Unlock()

	redo.logMap[blockHash] = log
}

func (redo *StorageRedo) RemoveLog(blockHash types.Hash) {
	redo.mu.Lock()
	defer redo.mu.Unlock()

	delete(redo.logMap, blockHash)
}

func (redo *StorageRedo) Rollback(snapshotHeight uint64) {
	if snapshotHeight != redo.snapshotHeight {
		redo.rollbackHeights = append(redo.rollbackHeights, snapshotHeight)
	} else {
		redo.snapshotHeight = 0
		redo.logMap = make(map[types.Hash][]byte)
	}

}

func (redo *StorageRedo) Id() types.Hash {
	return redo.id
}

func (redo *StorageRedo) Prepare() {
	if len(redo.rollbackHeights) > 0 {
		return
	}
	redo.flushingBatch.Reset()
	for blockHash, kvLog := range redo.logMap {
		redo.flushingBatch.Put(blockHash.Bytes(), kvLog)
	}

}

func (redo *StorageRedo) CancelPrepare() {
	redo.flushingBatch.Reset()
}

func (redo *StorageRedo) RedoLog() ([]byte, error) {
	var redoLog []byte
	if len(redo.rollbackHeights) > 0 {
		redoLog = make([]byte, 0, 1+8*len(redo.rollbackHeights))
		redoLog = append(redoLog, optRollback)

		for _, rollbackHeight := range redo.rollbackHeights {
			redoLog = append(redoLog, chain_utils.Uint64ToBytes(rollbackHeight)...)
		}
	} else {

		if redo.flushingBatch.Len() <= 0 {
			return nil, nil
		}

		redoLog = make([]byte, 0, 9+len(redo.flushingBatch.Dump()))

		redoLog = append(redoLog, optFlush)

		redoLog = append(redoLog, chain_utils.Uint64ToBytes(redo.snapshotHeight)...)

		redoLog = append(redoLog, redo.flushingBatch.Dump()...)
	}

	return redoLog, nil
}

func (redo *StorageRedo) Commit() error {
	if len(redo.rollbackHeights) > 0 {
		defer func() {
			redo.rollbackHeights = nil
		}()

		return redo.delete(redo.rollbackHeights)
	} else if redo.flushingBatch.Len() > 0 {
		defer func() {
			redo.flushingBatch.Reset()
		}()

		return redo.flush(redo.snapshotHeight, redo.flushingBatch)
	}

	return nil
}

func (redo *StorageRedo) PatchRedoLog(redoLog []byte) error {
	switch redoLog[0] {
	case optFlush:
		batch := new(leveldb.Batch)
		if err := batch.Load(redoLog[9:]); err != nil {
			return err
		}
		if batch.Len() <= 0 {
			return nil
		}
		return redo.flush(chain_utils.BytesToUint64(redoLog[1:9]), batch)
	case optRollback:
		currentPointer := 1
		redoLogLen := len(redoLog)

		snapshotHeights := make([]uint64, 0, redoLogLen-1/8)

		for currentPointer < redoLogLen {
			snapshotHeights = append(snapshotHeights, binary.BigEndian.Uint64(redoLog[currentPointer:currentPointer+8]))
			currentPointer += 8
		}

		return redo.delete(redo.rollbackHeights)
	}

	return nil
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

	fmt.Println("create bucket", snapshotHeight)

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

func (redo *StorageRedo) delete(snapshotHeights []uint64) error {
	tx, err := redo.store.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, snapshotHeight := range snapshotHeights {
		if err := tx.DeleteBucket(chain_utils.Uint64ToBytes(snapshotHeight)); err != nil {
			return err
		}
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
