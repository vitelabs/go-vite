package sender

import (
	"encoding/binary"
	"github.com/Shopify/sarama"
	"github.com/gin-gonic/gin/json"
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"sync"
	"time"
)

const (
	STOPPED = iota
	RUNNING
)

type message struct {
	MsgType string `json:"type"`
	Data    string `json:"data"`
	EventId uint64 `json:"eventId"`
}

type Producer struct {
	producerId uint8
	db         *leveldb.DB

	brokerList []string
	topic      string

	hasSend          uint64
	dbHasSend        uint64
	dbRecordInterval uint64

	termination chan int

	status     int
	statusLock sync.Mutex
	log        log15.Logger

	wg sync.WaitGroup

	kafkaProducer sarama.AsyncProducer
	chain         Chain
}

func NewProducerFromDb(producerId uint8, buf []byte, chain Chain, db *leveldb.DB) (*Producer, error) {
	producer := &Producer{}
	if dsErr := producer.Deserialize(buf); dsErr != nil {
		return nil, dsErr
	}

	if err := producer.init(producerId, chain, db); err != nil {
		return nil, err
	}
	return producer, nil
}

func NewProducer(producerId uint8, brokerList []string, topic string, chain Chain, db *leveldb.DB) (*Producer, error) {
	producer := &Producer{
		brokerList: brokerList,
		topic:      topic,
	}

	if err := producer.init(producerId, chain, db); err != nil {
		return nil, err
	}
	return producer, nil
}

func (producer *Producer) init(producerId uint8, chain Chain, db *leveldb.DB) error {
	producer.producerId = producerId

	hasSend, err := producer.getHasSend()
	if err != nil {
		return err
	}

	producer.hasSend = hasSend
	producer.dbHasSend = hasSend

	producer.chain = chain
	producer.db = db
	producer.dbRecordInterval = 100
}
func (producer *Producer) BrokerList() []string {
	return producer.brokerList
}

func (producer *Producer) Topic() string {
	return producer.topic
}

func (producer *Producer) IsSame(brokerList []string, topic string) bool {
	if producer.topic != topic ||
		len(brokerList) != len(producer.brokerList) {
		return false
	}

	tmpBrokerList := brokerList[:]
	for _, aBroker := range producer.brokerList {
		hasExist := false
		tmpBrokerListLen := len(tmpBrokerList)

		for i := 0; i < tmpBrokerListLen; i++ {
			if aBroker == tmpBrokerList[i] {
				tmpBrokerList = append(tmpBrokerList[:i], tmpBrokerList[i+1:]...)
				hasExist = true
			}
		}

		if !hasExist {
			return false
		}
	}
	return true
}

func (producer *Producer) Deserialize(buffer []byte) error {
	pb := &vitepb.Producer{}
	if err := proto.Unmarshal(buffer, pb); err != nil {
		return err
	}

	producer.topic = pb.Topic
	producer.brokerList = pb.BrokerList
	return nil
}

func (producer *Producer) Serialize() ([]byte, error) {
	pb := &vitepb.Producer{}
	pb.BrokerList = producer.brokerList
	pb.Topic = producer.topic

	return proto.Marshal(pb)
}

func (producer *Producer) Start() error {
	producer.statusLock.Lock()
	defer producer.statusLock.Unlock()
	if producer.status == RUNNING {
		return nil
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	kafkaProducer, err := sarama.NewAsyncProducer(producer.brokerList, config)
	if err != nil {
		return err
	}

	producer.kafkaProducer = &kafkaProducer
	producer.status = RUNNING
	producer.termination = make(chan int)

	go func() {
		defer producer.wg.Done()
		for {
			select {
			case producer.termination:
				tryCloseCount := 3
				closeCount := 0

				for ; closeCount < tryCloseCount; closeCount++ {
					closeErr := producer.kafkaProducer.Close()

					if closeErr != nil {
						producer.log.Error("kafkaProducer close failed, error is "+closeErr.Error(), "method", "Start")
					}
				}

				if closeCount == tryCloseCount {
					producer.log.Crit("kafkaProducer close failed", "method", "Start")
				}

				producer.kafkaProducer = nil
				return
			default:
				producer.send()
				time.Sleep(time.Second * 3)
			}
		}
	}()
	return nil
}

func (producer *Producer) Stop() {
	producer.statusLock.Lock()
	defer producer.statusLock.Unlock()
	if producer.status == STOPPED {
		return
	}

	producer.termination <- 1
	producer.wg.Wait()

	producer.status = STOPPED
}

func (producer *Producer) send() {
	start := producer.hasSend
	end := producer.chain.GetLatestBlockEventId()
	for i := start + 1; i <= end; i++ {
		if producer.hasSend > producer.dbHasSend &&
			producer.hasSend-producer.dbHasSend >= producer.dbRecordInterval {

			if err := producer.saveHasSend(); err != nil {
				producer.log.Error("saveHasSend failed, error is "+err.Error(), "method", "send")
			}

		}

		eventType, blockHashList, err := producer.chain.GetEvent(i)
		if err != nil {
			producer.log.Error("Get event failed, error is "+err.Error(), "method", "send")
			return
		}

		m := &message{
			EventId: i,
		}

		switch eventType {
		// AddAccountBlocksEvent     = byte(1)
		case byte(1):
			m.MsgType = "InsertAccountBlocks"
			var blocks []*ledger.AccountBlock
			for _, blockHash := range blockHashList {
				block, err := producer.chain.GetAccountBlockByHash(&blockHash)
				if err != nil {
					producer.log.Error("GetAccountBlockByHash failed, error is "+err.Error(), "method", "send")
					return
				}
				if block != nil {
					blocks = append(blocks, block)
				}
			}

			buf, jsonErr := json.Marshal(blocks)
			if jsonErr != nil {
				producer.log.Error("[InsertAccountBlocks] json.Marshal failed, error is "+jsonErr.Error(), "method", "send")
				return
			}
			m.Data += string(buf)

		// DeleteAccountBlocksEvent  = byte(2)
		case byte(2):
			m.MsgType = "DeleteAccountBlocks"

			buf, jsonErr := json.Marshal(blockHashList)
			if jsonErr != nil {
				producer.log.Error("[DeleteAccountBlocks] json.Marshal failed, error is "+jsonErr.Error(), "method", "send")
				return
			}
			m.Data = string(buf)

		// AddSnapshotBlocksEvent    = byte(3)
		case byte(3):
			m.MsgType = "InsertSnapshotBlocks"
			var blocks []*ledger.SnapshotBlock
			for _, blockHash := range blockHashList {
				block, err := producer.chain.GetSnapshotBlockByHash(&blockHash)
				if err != nil {
					producer.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "send")
					return
				}
				if block != nil {
					blocks = append(blocks, block)
				}
			}
			buf, jsonErr := json.Marshal(blocks)
			if jsonErr != nil {
				producer.log.Error("[InsertSnapshotBlocks] json.Marshal failed, error is "+jsonErr.Error(), "method", "send")
				return
			}
			m.Data += string(buf)

		// DeleteSnapshotBlocksEvent = byte(4)
		case byte(4):
			m.MsgType = "DeleteSnapshotBlocks"
			buf, jsonErr := json.Marshal(blockHashList)
			if jsonErr != nil {
				producer.log.Error("[DeleteSnapshotBlocks] json.Marshal failed, error is "+jsonErr.Error(), "method", "send")
				return
			}
			m.Data = string(buf)

		// No event
		default:
			producer.hasSend = i
			continue
		}

		sendErr := producer.sendMessage(m)
		if sendErr != nil {
			producer.log.Error("sendMessage failed, error is "+sendErr.Error(), "method", "send")
			return
		}

		producer.hasSend = i
	}
}

func (producer *Producer) saveHasSend() error {
	key := append([]byte{DBKP_PRODUCER_HAS_SEND}, producer.producerId)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, producer.hasSend)

	if err := producer.db.Put(key, buf, nil); err != nil {
		return err
	}

	producer.dbHasSend = producer.hasSend
	return nil
}

func (producer *Producer) getHasSend() (uint64, error) {
	key := append([]byte{DBKP_PRODUCER_HAS_SEND}, producer.producerId)

	value, err := producer.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}

	return binary.BigEndian.Uint64(value), nil
}

func (producer *Producer) sendMessage(msg *message) error {
	buf, jsonErr := json.Marshal(msg)
	if jsonErr != nil {
		return jsonErr
	}
	sMsg := &sarama.ProducerMessage{Topic: producer.topic, Value: sarama.StringEncoder(buf)}

	producer.kafkaProducer.Input() <- sMsg
	select {
	// success
	case <-producer.kafkaProducer.Successes():
		break
		// error
	case sendError := <-producer.kafkaProducer.Errors():
		producer.log.Error("kafka send failed, error is "+sendError.Error(), "method", "sendMessage")
		return sendError
	}
	return nil
}
