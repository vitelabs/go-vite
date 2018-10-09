package sender

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

var (
	DBK_TOTAL = []byte{byte(1)}

	DBKP_PRODUCER = byte(2)
)

type Indexer struct {
	db *leveldb.DB

	producers []*Producer
	total     uint64

	getLock sync.Mutex
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

func (indexer *Indexer) GetProducer(brokerList []string, topic string, version string) (*Producer, error) {
	indexer.getLock.Lock()
	defer indexer.getLock.Unlock()

	newProducer := NewProducer(brokerList, topic, version)
	for _, producer := range indexer.producers {
		if indexer.isSameProducer(producer, newProducer) {
			return producer, nil
		}
	}

	if writeErr := indexer.writeProducerToDb(newProducer); writeErr != nil {
		return nil, writeErr
	}

	indexer.producers = append(indexer.producers, newProducer)
	return newProducer, nil
}

func (indexer *Indexer) isSameProducer(producer1 *Producer, producer2 *Producer) bool {
	return true
}

func (indexer *Indexer) writeProducerToDb(producer *Producer) error {
	length := uint32(len(indexer.producers))

	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, length)

	key := append([]byte{DBKP_PRODUCER}, lengthBytes...)
	buf, sErr := producer.Serialize()
	if sErr != nil {
		return sErr
	}

	wErr := indexer.db.Put(key, buf, nil)
	return wErr
}

func (indexer *Indexer) readTotalFromDb() (uint64, error) {
	value, err := indexer.db.Get(DBK_TOTAL, nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(value), nil
}

func (indexer *Indexer) readProducersFromDb() ([]*Producer, error) {
	iter := indexer.db.NewIterator(util.BytesPrefix([]byte{byte(DBKP_PRODUCER)}), nil)
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
