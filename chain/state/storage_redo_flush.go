package chain_state

import (
	"encoding/binary"
	"github.com/boltdb/bolt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
)

func (redo *StorageRedo) Id() types.Hash {
	return redo.id
}

func (redo *StorageRedo) Prepare() {
	if len(redo.rollbackHeights) > 0 {
		return
	}

	latestSb := redo.chain.GetLatestSnapshotBlock()

	redo.flushingBatchMap = make(map[uint64]*leveldb.Batch, len(redo.snapshotLogMap))

	for snapshotHeight, snapshotLog := range redo.snapshotLogMap {
		if snapshotHeight > latestSb.Height || !snapshotLog.hasRedo {
			continue
		}

		batch, ok := redo.flushingBatchMap[snapshotHeight]
		if !ok || batch == nil {
			batch = new(leveldb.Batch)
			redo.flushingBatchMap[snapshotHeight] = batch
		}
		for blockHash, kvLog := range snapshotLog.log {
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
		redo.snapshotLogMap = make(map[uint64]*snapshotLog)
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
