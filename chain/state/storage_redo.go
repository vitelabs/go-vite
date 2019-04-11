package chain_state

import (
	"github.com/boltdb/bolt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"path"
	"sync"
)

const (
	optFlush    = 1
	optRollback = 2
)

type snapshotLog struct {
	log     map[types.Hash][]byte
	hasRedo bool
}

type StorageRedo struct {
	store *bolt.DB

	chain          Chain
	snapshotLogMap map[uint64]*snapshotLog

	currentSnapshotHeight uint64

	retainHeight uint64

	id types.Hash
	mu sync.RWMutex

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
		snapshotLogMap: make(map[uint64]*snapshotLog),
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

		redo.setSnapshot(snapshotHeight, redoLogMap, hasRedo)

		return nil
	})
}

func (redo *StorageRedo) NextSnapshot(nextSnapshotHeight uint64, confirmedBlocks []*ledger.AccountBlock) {

	sl := redo.snapshotLogMap[redo.currentSnapshotHeight]
	if !sl.hasRedo {
		redo.setSnapshot(nextSnapshotHeight, make(map[types.Hash][]byte), true)
		return
	}

	logMap := sl.log
	currentRedoLog := make(map[types.Hash][]byte, len(logMap))

	if len(logMap) > 0 {
		for _, confirmedBlock := range confirmedBlocks {
			if log, ok := logMap[confirmedBlock.Hash]; ok {
				currentRedoLog[confirmedBlock.Hash] = log
				delete(logMap, confirmedBlock.Hash)
			}
		}
	}

	redo.snapshotLogMap[redo.currentSnapshotHeight] = &snapshotLog{
		log:     currentRedoLog,
		hasRedo: sl.hasRedo,
	}

	redo.setSnapshot(nextSnapshotHeight, logMap, true)

}

func (redo *StorageRedo) HasRedo() bool {
	redo.mu.RLock()
	defer redo.mu.RUnlock()

	return redo.hasRedo()
}
func (redo *StorageRedo) QueryLog(snapshotHeight uint64) (map[types.Hash][]byte, bool, error) {
	redo.mu.RLock()
	if snapshotLog, ok := redo.snapshotLogMap[snapshotHeight]; ok {
		redo.mu.RUnlock()
		return snapshotLog.log, snapshotLog.hasRedo, nil
	}
	redo.mu.RUnlock()

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
	if !redo.hasRedo() {
		return
	}

	logMap := redo.snapshotLogMap[redo.currentSnapshotHeight].log
	logMap[blockHash] = log
}

func (redo *StorageRedo) RemoveLog(blockHash types.Hash) {
	redo.mu.Lock()
	defer redo.mu.Unlock()

	if !redo.hasRedo() {
		return
	}

	delete(redo.snapshotLogMap[redo.currentSnapshotHeight].log, blockHash)
}

func (redo *StorageRedo) Rollback(snapshotHeight uint64) {
	redo.rollbackHeights = append(redo.rollbackHeights, snapshotHeight)
}

func (redo *StorageRedo) setSnapshot(snapshotHeight uint64, redoLog map[types.Hash][]byte, hasRedo bool) {

	redo.snapshotLogMap[snapshotHeight] = &snapshotLog{
		log:     redoLog,
		hasRedo: hasRedo,
	}

	redo.currentSnapshotHeight = snapshotHeight
}

func (redo *StorageRedo) hasRedo() bool {
	return redo.snapshotLogMap[redo.currentSnapshotHeight].hasRedo

}
