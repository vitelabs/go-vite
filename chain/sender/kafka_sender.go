package sender

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"sync"
)

const (
	DBKP_PRODUCER          = byte(1)
	DBKP_PRODUCER_HAS_SEND = byte(2)
)

type KafkaSender struct {
	producers    []*Producer
	runProducers []*Producer

	chain Chain
	db    *leveldb.DB

	lock sync.Mutex
	log  log15.Logger
}

func NewKafkaSender(chain Chain, dirName string) (*KafkaSender, error) {
	// create directory
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		os.Mkdir(dirName, 0755)
	}

	sender := &KafkaSender{
		//producer: producer,
		chain: chain,
		log:   log15.New("module", "chain/sender"),
	}

	db, openDbErr := leveldb.OpenFile(dirName, nil)

	if openDbErr != nil {
		return nil, openDbErr
	}

	sender.db = db

	producers, readPuErr := sender.readProducersFromDb()
	if readPuErr != nil {
		return nil, readPuErr
	}

	sender.producers = producers

	return sender, nil
}

func (sender *KafkaSender) Start(brokerList []string, topic string) error {
	sender.lock.Lock()
	defer sender.lock.Unlock()

	producer, err := sender.getProducer(brokerList, topic)
	if err != nil {
		return err
	}

	for _, runProducer := range sender.runProducers {
		if runProducer.IsSame(brokerList, topic) {
			// has run
			return nil
		}
	}

	if startErr := producer.Start(); startErr != nil {
		return startErr
	}

	sender.runProducers = append(sender.runProducers, producer)

	return nil
}

func (sender *KafkaSender) Stop(brokerList []string, topic string) {
	sender.lock.Lock()
	defer sender.lock.Unlock()

	for index, runProducer := range sender.runProducers {
		if runProducer.IsSame(brokerList, topic) {
			// has run
			runProducer.Stop()
			sender.runProducers = append(sender.runProducers[:index], sender.runProducers[index+1:]...)
		}
	}
}

func (sender *KafkaSender) StopAll() {
	sender.lock.Lock()
	defer sender.lock.Unlock()

	for _, runProducer := range sender.runProducers {
		runProducer.Stop()
	}
}

func (sender *KafkaSender) Producers() []*Producer {
	return sender.producers
}

func (sender *KafkaSender) RunProducers() []*Producer {
	return sender.producers
}

func (sender *KafkaSender) getProducer(brokerList []string, topic string) (*Producer, error) {
	for _, producer := range sender.producers {
		if producer.IsSame(brokerList, topic) {
			return producer, nil
		}
	}

	newProducer, newErr := NewProducer(byte(len(sender.producers)+1), brokerList, topic, sender.chain, sender.db)
	if newErr != nil {
		return nil, newErr
	}

	if writeErr := sender.writeProducerToDb(newProducer); writeErr != nil {
		return nil, writeErr
	}

	sender.producers = append(sender.producers, newProducer)
	return newProducer, nil
}

func (sender *KafkaSender) writeProducerToDb(producer *Producer) error {
	key := append([]byte{DBKP_PRODUCER}, producer.producerId)
	buf, sErr := producer.Serialize()

	if sErr != nil {
		return sErr
	}

	wErr := sender.db.Put(key, buf, nil)
	return wErr
}

func (sender *KafkaSender) readProducersFromDb() ([]*Producer, error) {
	iter := sender.db.NewIterator(util.BytesPrefix([]byte{byte(DBKP_PRODUCER)}), nil)
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
		producerId := uint8(iter.Key()[1])

		producer, err := NewProducerFromDb(producerId, iter.Value(), sender.chain, sender.db)

		if err != nil {
			return nil, err
		}
		producers = append(producers, producer)
	}

	return producers, nil
}
