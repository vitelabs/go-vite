package chain_state

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
)

type multiVersionDB struct {
	db *leveldb.DB
}

func newMultiVersionDB() *multiVersionDB {
	return &multiVersionDB{}
}

// TODO valueId
func (mvDB *multiVersionDB) Insert(keyList [][]byte, valueList [][]byte) error {

	batch := new(leveldb.Batch)
	for index, key := range keyList {
		prevValueId, err := mvDB.getValueId(key)
		if err != nil {
			return err
		}

		valueId := uint64(100)

		// update key
		batch.Put(key, chain_dbutils.Uint64ToFixedBytes(valueId))

		// insert value
		valueIdKey := chain_dbutils.CreateValueIdKey(valueId)
		batch.Put(valueIdKey, append(chain_dbutils.Uint64ToFixedBytes(prevValueId), valueList[index]...))

	}

	if err := mvDB.db.Write(batch, nil); err != nil {
		return err
	}
	return nil
}

func (mvDB *multiVersionDB) getValueId(key []byte) (uint64, error) {
	valueIdBytes, err := mvDB.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}

		return 0, err
	}

	return chain_dbutils.FixedBytesToUint64(valueIdBytes), nil
}
