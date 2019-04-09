package chain_state

import (
	"encoding/binary"
	"path"
	"sync"

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
	store *bolt.DB
	//logMap         map[types.Hash][]byte
	//snapshotHeight uint64

	snapshotLogMap        map[uint64]map[types.Hash][]byte
	currentSnapshotHeight uint64

	retainHeight uint64
	hasRedo      bool

	id types.Hash
	mu sync.RWMutex

	flushingBatchMap map[uint64]*leveldb.Batch
	rollbackHeights  []uint64
}

func NewStorageRedo(chainDir string) (*StorageRedo, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("state_db_kv_redo")))

	store, err := bolt.Open(path.Join(chainDir, "kv_redo"), 0600, nil)
	if err != nil {
		return nil, err
	}
	return &StorageRedo{
		store:          store,
		snapshotLogMap: make(map[uint64]map[types.Hash][]byte),
		//logMap:       make(map[types.Hash][]byte),
		retainHeight: 600,
		id:           id,

		flushingBatchMap: make(map[uint64]*leveldb.Batch),
	}, nil
}

func (redo *StorageRedo) Close() error {
	if err := redo.store.Close(); err != nil {
		return err
	}
	redo.store = nil
	return nil
}

func (redo *StorageRedo) SetSnapshot(snapshotHeight uint64, redoLog map[types.Hash][]byte, hasRedo bool) {
	//redo.logMap = redoLog
	//
	//if redo.logMap == nil {
	//	redo.logMap = make(map[types.Hash][]byte)
	//}
	//redo.snapshotHeight = snapshotHeight
	if redoLog == nil {
		redo.snapshotLogMap[snapshotHeight] = make(map[types.Hash][]byte)
	} else {
		redo.snapshotLogMap[snapshotHeight] = redoLog
	}
	redo.currentSnapshotHeight = snapshotHeight
	redo.hasRedo = hasRedo
}

func (redo *StorageRedo) QueryLog(snapshotHeight uint64) (map[types.Hash][]byte, bool, error) {
	//if snapshotHeight == redo.snapshotHeight {
	//	return redo.logMap, true, nil
	//}

	if logMap, ok := redo.snapshotLogMap[snapshotHeight]; ok {
		return logMap, true, nil
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

	redo.snapshotLogMap[redo.currentSnapshotHeight][blockHash] = log
	//redo.logMap[blockHash] = log
}

func (redo *StorageRedo) RemoveLog(blockHash types.Hash) {
	redo.mu.Lock()
	defer redo.mu.Unlock()

	delete(redo.snapshotLogMap[redo.currentSnapshotHeight], blockHash)
}

func (redo *StorageRedo) PrepareRollback(snapshotHeight uint64) {
	redo.rollbackHeights = append(redo.rollbackHeights, snapshotHeight)
}

func (redo *StorageRedo) Id() types.Hash {
	return redo.id
}

func (redo *StorageRedo) Prepare() {
	if len(redo.rollbackHeights) > 0 {
		return
	}

	redo.flushingBatchMap = make(map[uint64]*leveldb.Batch, len(redo.snapshotLogMap))

	for snapshotHeight, logMap := range redo.snapshotLogMap {
		batch, ok := redo.flushingBatchMap[snapshotHeight]
		if !ok || batch == nil {
			batch = new(leveldb.Batch)
			redo.flushingBatchMap[snapshotHeight] = batch
		}
		for blockHash, kvLog := range logMap {
			batch.Put(blockHash.Bytes(), kvLog)
		}
	}

}

func (redo *StorageRedo) CancelPrepare() {
	redo.flushingBatchMap = make(map[uint64]*leveldb.Batch, 0)
}

func (redo *StorageRedo) RedoLog() ([]byte, error) {
	var redoLog []byte
	if len(redo.rollbackHeights) > 0 {
		redoLog = make([]byte, 0, 1+8*len(redo.rollbackHeights))
		redoLog = append(redoLog, optRollback)

		for _, rollbackHeight := range redo.rollbackHeights {
			redoLog = append(redoLog, chain_utils.Uint64ToBytes(rollbackHeight)...)
		}
	} else if len(redo.flushingBatchMap) > 0 {

		redoLogSize := 1
		for _, batch := range redo.flushingBatchMap {
			redoLogSize += 12 + len(batch.Dump())
		}

		redoLog = make([]byte, 0, redoLogSize)

		redoLog = append(redoLog, optFlush)

		for snapshotHeight, batch := range redo.flushingBatchMap {
			redoLog = append(redoLog, chain_utils.Uint64ToBytes(snapshotHeight)...)

			batchLen := len(batch.Dump())
			batchLenBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(batchLenBytes, uint32(batchLen))

			redoLog = append(redoLog, batchLenBytes...)
			redoLog = append(redoLog, batch.Dump()...)
		}

	}

	return redoLog, nil
}

func (redo *StorageRedo) Commit() error {
	if len(redo.rollbackHeights) > 0 {
		defer func() {
			redo.rollbackHeights = nil
		}()

		return redo.delete(redo.rollbackHeights)
	} else if len(redo.flushingBatchMap) > 0 {
		defer func() {
			redo.flushingBatchMap = make(map[uint64]*leveldb.Batch)
		}()

		for snapshotHeight, batch := range redo.flushingBatchMap {
			if err := redo.flush(snapshotHeight, batch); err != nil {
				return err
			}
		}
	}

	return nil
}

func (redo *StorageRedo) PatchRedoLog(redoLog []byte) error {
	switch redoLog[0] {
	case optFlush:

		redoLogLen := len(redoLog)
		currentPointer := 1

		for currentPointer < redoLogLen {
			batch := new(leveldb.Batch)

			snapshotHeight := chain_utils.BytesToUint64(redoLog[currentPointer : currentPointer+8])
			currentPointer += 8

			size := binary.BigEndian.Uint32(redoLog[currentPointer : currentPointer+4])
			currentPointer += 4

			batchBytes := redoLog[currentPointer : currentPointer+int(size)]
			currentPointer += int(size)

			batch.Load(batchBytes)

			if err := redo.flush(snapshotHeight, batch); err != nil {
				return err
			}

		}

		return nil
	case optRollback:
		currentPointer := 1
		redoLogLen := len(redoLog)

		snapshotHeights := make([]uint64, 0, (redoLogLen-1)/8)

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

	// add
	batch.Replay(NewBatchFlush(bu))

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
		if _, ok := redo.snapshotLogMap[snapshotHeight]; ok {
			delete(redo.snapshotLogMap, snapshotHeight)
			continue
		}

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
