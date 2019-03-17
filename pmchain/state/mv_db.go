package chain_state

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
	"github.com/vitelabs/go-vite/pmchain/pending"
	"path"
)

type multiVersionDB struct {
	db *leveldb.DB

	pending *chain_pending.MemDB
}

func newMultiVersionDB(chainDir string) (*multiVersionDB, error) {

	dbDir := path.Join(chainDir, "state")

	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}

	return &multiVersionDB{
		db:      db,
		pending: chain_pending.NewMemDB(),
	}, nil
}

func (mvDB *multiVersionDB) Destroy() error {
	if err := mvDB.db.Close(); err != nil {
		return err
	}

	mvDB.db = nil

	mvDB.pending.Clean()
	mvDB.pending = nil

	return nil
}

// TODO valueId
func (mvDB *multiVersionDB) Insert(block *ledger.AccountBlock, keyList [][]byte, valueList [][]byte) error {

	prevValueIdList := make([]uint64, len(keyList))
	for index, key := range keyList {
		prevValueId, err := mvDB.getLatestValueId(key)
		if err != nil {
			return err
		}
		prevValueIdList[index] = prevValueId

		valueId := uint64(100)

		// update key
		mvDB.pending.Put(&block.Hash, key, chain_dbutils.Uint64ToFixedBytes(valueId))

		// insert value
		valueIdKey := chain_dbutils.CreateValueIdKey(valueId)
		mvDB.pending.Put(&block.Hash, valueIdKey, valueList[index])
	}

	// insert undo log
	if err := mvDB.writeUndoLog(block, keyList, prevValueIdList); err != nil {
		return err
	}
	return nil
}

func (mvDB *multiVersionDB) Flush(blockHashList []*types.Hash) error {
	batch := new(leveldb.Batch)

	keyList, valueList := mvDB.pending.GetByBlockHashList(blockHashList)
	for index, key := range keyList {
		batch.Put(key, valueList[index])
	}

	if err := mvDB.db.Write(batch, nil); err != nil {
		return err
	}
	mvDB.pending.DeleteByBlockHashList(blockHashList)
	return nil
}

func (mvDB *multiVersionDB) DeletePendingByBlockHash(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		mvDB.pending.DeleteByBlockHash(&block.Hash)
	}
}

func (mvDB *multiVersionDB) writeUndoLog(block *ledger.AccountBlock, keyList [][]byte, valueIdList []uint64) error {
	undoLog := make([]byte, 0, len(valueIdList)*16)
	for index, key := range keyList {
		keyId, err := mvDB.getKeyId(key)
		if err != nil {
			return err
		}
		undoLog = append(undoLog, chain_dbutils.Uint64ToFixedBytes(keyId)...)
		undoLog = append(undoLog, chain_dbutils.Uint64ToFixedBytes(valueIdList[index])...)
	}

	undoKey := chain_dbutils.CreateStateUndoKey(1, block.Height)
	mvDB.pending.Put(&block.Hash, undoKey, undoLog)
	return nil
}

func (mvDB *multiVersionDB) GetUndoLog(accountId uint64, height uint64) []byte {
	return nil
}

func (mvDB *multiVersionDB) DeleteUndoLog(accountId uint64, height uint64) []byte {
	return nil
}

func (mvDB *multiVersionDB) getKeyId(key []byte) (uint64, error) {
	keyIdBytes, err := mvDB.db.Get(chain_dbutils.CreateKeyIdKey(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}

	return chain_dbutils.DeserializeUint64(keyIdBytes), nil
}

func (mvDB *multiVersionDB) getLatestValueId(key []byte) (uint64, error) {
	valueIdBytes := make([]byte, 0, 8)

	var ok bool
	if valueIdBytes, ok = mvDB.pending.Get(key); !ok {
		var err error
		valueIdBytes, err = mvDB.db.Get(key, nil)
		if err != nil {
			if err == leveldb.ErrNotFound {
				return 0, nil
			}

			return 0, err
		}
	}

	return chain_dbutils.FixedBytesToUint64(valueIdBytes), nil
}
