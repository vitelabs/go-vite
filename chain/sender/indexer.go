package sender

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	DBK_TOTAL = []byte{byte(1)}

	DBKP_PRODUCER_UNIT = byte(2)
)

type Indexer struct {
	db *leveldb.DB

	producers []*Producer
	total     uint64
}

func NewIndexer(dirName string) (*Indexer, error) {
	db, openDbErr := leveldb.OpenFile(dirName, nil)

	if openDbErr != nil {
		return nil, openDbErr
	}

	indexer := &Indexer{
		db: db,
	}

	total, readTotalErr := indexer.readTotalFromDb()
	if readTotalErr != nil {
		return nil, readTotalErr
	}

	indexer.total = total

	producers, readPuErr := indexer.readProducersFromDb()
	if readPuErr != nil {
		return nil, readPuErr
	}
	indexer.producers = producers

	return indexer, nil
}

func (indexer *Indexer) readTotalFromDb() (uint64, error) {
	value, err := indexer.db.Get(DBK_TOTAL, nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(value), nil
}

func (indexer *Indexer) readProducersFromDb() ([]*Producer, error) {
	iter := indexer.db.NewIterator(util.BytesPrefix([]byte{byte(DBKP_PRODUCER_UNIT)}), nil)
	defer iter.Release()

	var producers []*Producer
	for {
		iterOk := iter.Next()
		if !iterOk {
			if iterErr := iter.Error(); iterErr != nil && iterErr != leveldb.ErrNotFound {
				return nil, iterErr
			}
			break
		}

		producerUnit := &Producer{}
		if dsErr := producerUnit.Deserialize(iter.Value()); dsErr != nil {
			return nil, dsErr
		}

		producers = append(producers, producerUnit)
	}

	return producers, nil
}
