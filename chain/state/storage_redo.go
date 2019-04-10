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

	chain          Chain
	snapshotLogMap map[uint64]map[types.Hash][]byte

	currentSnapshotHeight uint64

	retainHeight uint64

	id types.Hash
	mu sync.RWMutex

	hasRedo          bool
	flushingBatchMap map[uint64]*leveldb.Batch
	rollbackHeights  []uint64
}

func NewStorageRedo(chain Chain, chainDir string) (*StorageRedo, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("state_db_kv_redo")))

	store, err := bolt.Open(path.Join(chainDir, "kv_redo"), 0600, nil)
	if err != nil {
		return nil, err
	}
	redo := &StorageRedo{
		chain:          chain,
		store:          store,
		snapshotLogMap: make(map[uint64]map[types.Hash][]byte),
		retainHeight:   1200,
		id:             id,

		flushingBatchMap: make(map[uint64]*leveldb.Batch),
	}

	height := uint64(0)

	latestSnapshotBlock, err := chain.QueryLatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	if latestSnapshotBlock != nil {
		height = latestSnapshotBlock.Height
	}

	if err := redo.ResetSnapshot(height + 1); err != nil {
		return nil, err
	}
	return redo, nil
}

func (redo *StorageRedo) Close() error {
	if err := redo.store.Close(); err != nil {
		return err
	}
	redo.store = nil
	return nil
}

func (redo *StorageRedo) ResetSnapshot(snapshotHeight uint64) error {
	return redo.store.Update(func(tx *bolt.Tx) error {
		redoLogMap := make(map[types.Hash][]byte)

		bu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight))

		hasRedo := true
		if bu == nil {
			prevBu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight - 1))
			if prevBu == nil {
				hasRedo = false
			}

			var err error
			bu, err = tx.CreateBucket(chain_utils.Uint64ToBytes(snapshotHeight))
			if err != nil {
				return err
			}
		} else {
			c := bu.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				hash, err := types.BytesToHash(k)
				if err != nil {
					return err
				}
				redoLogMap[hash] = v
			}

		}

		if hasRedo {
			redo.SetSnapshot(snapshotHeight, redoLogMap)
		} else {
			redo.SetSnapshot(snapshotHeight, nil)
		}

		return nil
	})
}

func (redo *StorageRedo) SetSnapshot(snapshotHeight uint64, redoLog map[types.Hash][]byte) {

	//redo.snapshotLogMap[snapshotHeight] = make(map[types.Hash][]byte)

	redo.snapshotLogMap[snapshotHeight] = redoLog

	redo.currentSnapshotHeight = snapshotHeight
}

func (redo *StorageRedo) HasRedo() bool {
	return redo.snapshotLogMap[redo.currentSnapshotHeight] != nil
}
func (redo *StorageRedo) QueryLog(snapshotHeight uint64) (map[types.Hash][]byte, bool, error) {
	if logMap, ok := redo.snapshotLogMap[snapshotHeight]; ok {
		return logMap, logMap != nil, nil
	}

	logMap := make(map[types.Hash][]byte)

	hasRedo := true
	err := redo.store.View(func(tx *bolt.Tx) error {
		bu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight))
		if bu == nil {
			hasRedo = false
			return nil
		}

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

func (redo *StorageRedo) AddLog(blockHash types.Hash, log []byte) {
	redo.mu.Lock()
	defer redo.mu.Unlock()
	if !redo.HasRedo() {
		return
	}

	redo.snapshotLogMap[redo.currentSnapshotHeight][blockHash] = log
}

func (redo *StorageRedo) RemoveLog(blockHash types.Hash) {
	redo.mu.Lock()
	defer redo.mu.Unlock()

	if !redo.HasRedo() {
		return
	}

	delete(redo.snapshotLogMap[redo.currentSnapshotHeight], blockHash)
}

func (redo *StorageRedo) Rollback(snapshotHeight uint64) {
	redo.rollbackHeights = append(redo.rollbackHeights, snapshotHeight)
}

func (redo *StorageRedo) Id() types.Hash {
	return redo.id
}

func (redo *StorageRedo) Prepare() {
	if len(redo.rollbackHeights) > 0 {
		return
	}

	latestSb := redo.chain.GetLatestSnapshotBlock()

	redo.flushingBatchMap = make(map[uint64]*leveldb.Batch, len(redo.snapshotLogMap))

	for snapshotHeight, logMap := range redo.snapshotLogMap {
		if snapshotHeight > latestSb.Height {
			continue
		}

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

		// FOR DEBUG
		//fmt.Println("storage redo log start")
		for _, rollbackHeight := range redo.rollbackHeights {
			redoLog = append(redoLog, chain_utils.Uint64ToBytes(rollbackHeight)...)

			// FOR DEBUG
			//fmt.Println("storage redo ", rollbackHeight)
		}
		// FOR DEBUG
		//fmt.Println("storage redo log end")

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
	defer func() {
		redo.rollbackHeights = nil
		redo.flushingBatchMap = make(map[uint64]*leveldb.Batch)

		current := redo.snapshotLogMap[redo.currentSnapshotHeight]
		redo.snapshotLogMap = make(map[uint64]map[types.Hash][]byte)
		redo.snapshotLogMap[redo.currentSnapshotHeight] = current

	}()

	if len(redo.rollbackHeights) > 0 {

		return redo.delete(redo.rollbackHeights)
	} else {
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

		return redo.delete(snapshotHeights)
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
	if len(snapshotHeights) <= 0 {
		return nil
	}
	tx, err := redo.store.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, snapshotHeight := range snapshotHeights {
		tx.DeleteBucket(chain_utils.Uint64ToBytes(snapshotHeight))
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
