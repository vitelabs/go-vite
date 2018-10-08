package sender

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/vitepb"
)

var (
	DBK_TOTAL = []byte{byte(1)}

	DBKP_PRODUCER_UNIT = byte(2)
)

type Indexer struct {
	db *leveldb.DB

	producerUnits []*ProducerUnit
	total         uint64
}

type ProducerUnit struct {
	brokerList []string
	topic      string
	version    string

	hasSend uint64
}

func (producerUnit *ProducerUnit) Deserialize(buffer []byte) error {
	pb := &vitepb.ProducerUnit{}
	if err := proto.Unmarshal(buffer, pb); err != nil {
		return err
	}
	producerUnit.topic = pb.Topic
	producerUnit.brokerList = pb.BrokerList
	producerUnit.version = pb.Version
	producerUnit.hasSend = pb.HasSend
	return nil
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

	producerUnits, readPuErr := indexer.readProducerUnitsFromDb()
	if readPuErr != nil {
		return nil, readPuErr
	}
	indexer.producerUnits = producerUnits

	return indexer, nil
}

func (indexer *Indexer) readTotalFromDb() (uint64, error) {
	value, err := indexer.db.Get(DBK_TOTAL, nil)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(value), nil
}

func (indexer *Indexer) readProducerUnitsFromDb() ([]*ProducerUnit, error) {
	iter := indexer.db.NewIterator(util.BytesPrefix([]byte{byte(DBKP_PRODUCER_UNIT)}), nil)
	defer iter.Release()

	var producerUnits []*ProducerUnit
	for {
		iterOk := iter.Next()
		if !iterOk {
			if iterErr := iter.Error(); iterErr != nil && iterErr != leveldb.ErrNotFound {
				return nil, iterErr
			}
			break
		}

		producerUnit := &ProducerUnit{}
		if dsErr := producerUnit.Deserialize(iter.Value()); dsErr != nil {
			return nil, dsErr
		}

		producerUnits = append(producerUnits, producerUnit)
	}

	return producerUnits, nil
}
