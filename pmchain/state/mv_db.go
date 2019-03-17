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
func (mvDB *multiVersionDB) Insert(blockHash *types.Hash, keyList [][]byte, valueList [][]byte) error {

	for index, key := range keyList {
		prevValueId, err := mvDB.getValueId(key)
		if err != nil {
			return err
		}

		valueId := uint64(100)

		// update key
		mvDB.pending.Put(blockHash, key, chain_dbutils.Uint64ToFixedBytes(valueId))

		// insert value
		valueIdKey := chain_dbutils.CreateValueIdKey(valueId)
		mvDB.pending.Put(blockHash, valueIdKey, append(chain_dbutils.Uint64ToFixedBytes(prevValueId), valueList[index]...))

	}

	return nil
}

func (mvDB *multiVersionDB) Flush(blockHashList []*types.Hash) ([][]byte, error) {
	batch := new(leveldb.Batch)

	keyList, valueList := mvDB.pending.GetByBlockHashList(blockHashList)

	for index, key := range keyList {
		batch.Put(key, valueList[index])
	}

	if err := mvDB.db.Write(batch, nil); err != nil {
		return nil, err
	}
	mvDB.pending.DeleteByBlockHashList(blockHashList)
	return keyList, nil
}

func (mvDB *multiVersionDB) DeletePendingByBlockHash(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		mvDB.pending.DeleteByBlockHash(&block.Hash)
	}
}

func (mvDB *multiVersionDB) getValueId(key []byte) (uint64, error) {
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
