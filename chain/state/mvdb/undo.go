package mvdb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
)

func (mvDB *MultiVersionDB) Undo(blockHashList []*types.Hash) error {
	// clean pending
	mvDB.pending.Clean()

	batch := new(leveldb.Batch)

	// calculate undo
	undo := make(map[uint64]uint64)

	for _, blockHash := range blockHashList {
		undoLog, err := mvDB.GetUndoLog(blockHash)
		if err != nil {
			return err
		}

		mvDB.ParseUndoLog(undoLog, func(keyId uint64, valueId uint64) {
			if _, ok := undo[keyId]; !ok {
				undo[keyId] = valueId
			}
		})

		// delete undo log list
		mvDB.deleteUndoLog(batch, blockHash)
	}

	// set key index
	for keyId, valueId := range undo {
		if valueId <= 0 {

		} else {
			mvDB.updateKeyIdIndex(batch, keyId, valueId)
		}
	}

	return mvDB.db.Write(batch, nil)
}

func (mvDB *MultiVersionDB) GetUndoLog(blockHash *types.Hash) ([]byte, error) {
	return mvDB.db.Get(chain_utils.CreateStateUndoKey(blockHash), nil)
}

// positive sequence undo log list(from low to high)
func (mvDB *MultiVersionDB) PosSeqUndoLogListToOperations(undoLogList [][]byte) map[uint64]uint64 {
	size := 0
	for _, undoLog := range undoLogList {
		size += len(undoLog) / 8
	}

	operations := make(map[uint64]uint64, size)
	for _, undoLog := range undoLogList {
		currentPointer := 0
		undoLogLen := len(undoLog)
		for currentPointer < undoLogLen {
			undoKeyId := chain_utils.FixedBytesToUint64(undoLog[currentPointer : currentPointer+4])
			undoValueId := chain_utils.FixedBytesToUint64(undoLog[currentPointer+4 : currentPointer+8])

			if _, ok := operations[undoKeyId]; !ok {
				operations[undoKeyId] = undoValueId
			}

			currentPointer += 8
		}
	}

	return operations
}
func (mvDB *MultiVersionDB) ParseUndoLog(undoLog []byte, processor func(keyId uint64, valueId uint64)) {
	currentPointer := 0
	undoLogLen := len(undoLog)
	for currentPointer < undoLogLen {
		undoKeyId := chain_utils.FixedBytesToUint64(undoLog[currentPointer : currentPointer+4])
		undoValueId := chain_utils.FixedBytesToUint64(undoLog[currentPointer+4 : currentPointer+8])

		processor(undoKeyId, undoValueId)

		currentPointer += 8
	}
}

func (mvDB *MultiVersionDB) writeUndoLog(blockHash *types.Hash, keyList [][]byte, valueIdList []uint64) error {
	undoLog := make([]byte, 0, len(valueIdList)*16)
	for index, key := range keyList {
		keyId, err := mvDB.GetKeyId(key)
		if err != nil {
			return err
		}
		undoLog = append(undoLog, chain_utils.Uint64ToFixedBytes(keyId)...)
		undoLog = append(undoLog, chain_utils.Uint64ToFixedBytes(valueIdList[index])...)
	}

	undoKey := chain_utils.CreateStateUndoKey(blockHash)
	mvDB.pending.Put(blockHash, undoKey, undoLog)
	return nil
}

func (mvDB *MultiVersionDB) deleteUndoLog(batch *leveldb.Batch, blockHash *types.Hash) {
	batch.Delete(chain_utils.CreateStateUndoKey(blockHash))
}
