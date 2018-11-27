package sender

import (
	"encoding/binary"
	"encoding/json"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"github.com/vitelabs/go-vite/vm_context"
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

type MqSnapshotContentItem struct {
	Start *ledger.HashHeight `json:"start"`
	End   *ledger.HashHeight `json:"end"`
}

type MqSnapshotContent map[types.Address]*MqSnapshotContentItem

type MqSnapshotBlock struct {
	*ledger.SnapshotBlock
	MqSnapshotContent MqSnapshotContent `json:"snapshotContent"`
	Producer          types.Address     `json:"producer"`
	Timestamp         int64             `json:"timestamp"`
}

type MqAccountBlock struct {
	ledger.AccountBlock

	Balance     *big.Int      `json:"balance"`
	FromAddress types.Address `json:"fromAddress"`
	Timestamp   int64         `json:"timestamp"`

	ParsedData string `json:"parsedData"`
	SendData   []byte `json:"sendData"`
}

type Producer struct {
	producerId uint8
	db         *leveldb.DB

	brokerList []string
	topic      string

	hasSendLock      sync.RWMutex
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
	concurrency   uint64

	sendWg sync.WaitGroup
}

func NewProducerFromDb(producerId uint8, buf []byte, chain Chain, db *leveldb.DB) (*Producer, error) {
	producer := &Producer{}
	if dsErr := producer.Deserialize(buf); dsErr != nil {
		return nil, dsErr
	}

	if err := producer.init(producerId, chain, db); err != nil {
		return nil, err
	}
	producer.log = log15.New("module", "sender/producer")
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
	producer.log = log15.New("module", "sender/producer")
	return producer, nil
}

func (producer *Producer) init(producerId uint8, chain Chain, db *leveldb.DB) error {
	producer.producerId = producerId
	producer.concurrency = 100

	producer.chain = chain
	producer.db = db
	producer.dbRecordInterval = 100

	hasSend, err := producer.getHasSend()
	if err != nil {
		return err
	}

	producer.hasSendLock.Lock()

	producer.hasSend = hasSend
	producer.dbHasSend = hasSend

	producer.hasSendLock.Unlock()

	return nil
}

func (producer *Producer) SetHasSend(hasSend uint64) {
	producer.hasSendLock.Lock()
	defer producer.hasSendLock.Unlock()

	producer.hasSend = hasSend
	producer.saveHasSend()
}

func (producer *Producer) ProducerId() uint8 {
	return producer.producerId
}

func (producer *Producer) BrokerList() []string {
	return producer.brokerList
}

func (producer *Producer) Topic() string {
	return producer.topic
}

func (producer *Producer) HasSend() uint64 {
	return producer.hasSend
}

func (producer *Producer) Status() int {
	return producer.status
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
				break
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

	producer.kafkaProducer = kafkaProducer
	producer.status = RUNNING
	producer.termination = make(chan int)

	producer.wg.Add(1)
	common.Go(func() {
		defer producer.wg.Done()
		for {
			select {
			case <-producer.termination:
				tryCloseCount := 3
				closeCount := 0

				for ; closeCount < tryCloseCount; closeCount++ {
					closeErr := producer.kafkaProducer.Close()

					if closeErr != nil {
						producer.log.Error("kafkaProducer close failed, error is "+closeErr.Error(), "method", "Start")
					} else {
						producer.kafkaProducer = nil
						return
					}
				}

				if closeCount == tryCloseCount {
					producer.log.Crit("kafkaProducer close failed", "method", "Start")
				}
			default:
				producer.send()
				time.Sleep(time.Millisecond * 500)
			}
		}
	})
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

func (producer *Producer) getParsedData(block *ledger.AccountBlock) (string, error) {
	if len(block.Data) <= 0 {
		return "", nil
	}

	switch block.ToAddress.String() {
	case types.AddressMintage.String():
		tokenInfo := new(types.TokenInfo)
		token := abi.ABIMintage.UnpackVariable(tokenInfo, abi.VariableNameMintage, block.Data)
		tokenBytes, err := json.Marshal(token)
		return string(tokenBytes), err
	}

	return "", nil
}

func (producer *Producer) send() {
	producer.hasSendLock.Lock()
	defer producer.hasSendLock.Unlock()

	start := producer.hasSend
	end, err := producer.chain.GetLatestBlockEventId()
	if err != nil {
		producer.log.Error("GetLatestBlockEventId failed, error is "+err.Error(), "method", "send")
		return
	}

	defer func() {
		if producer.hasSend > producer.dbHasSend {
			if err := producer.saveHasSend(); err != nil {
				producer.log.Error("saveHasSend failed, error is "+err.Error(), "method", "send")
			}
		}
	}()

	for i := start; i < end; {
		if producer.hasSend > producer.dbHasSend &&
			producer.hasSend-producer.dbHasSend >= producer.dbRecordInterval {
			if err := producer.saveHasSend(); err != nil {
				producer.log.Error("saveHasSend failed, error is "+err.Error(), "method", "send")
			}

		}

		var msgList []*message

		j := i + 1
		for ; j-i <= producer.concurrency && j <= end; j++ {
			eventType, blockHashList, err := producer.chain.GetEvent(j)
			if err != nil {
				producer.log.Error("Get event failed, error is "+err.Error(), "method", "send")
				return
			}

			m := &message{
				EventId: j,
			}

			switch eventType {
			// AddAccountBlocksEvent     = byte(1)
			case byte(1):
				m.MsgType = "InsertAccountBlocks"
				var blocks []*MqAccountBlock
				for _, blockHash := range blockHashList {
					block, err := producer.chain.GetAccountBlockByHash(&blockHash)
					if err != nil {
						producer.log.Error("GetAccountBlockByHash failed, error is "+err.Error(), "method", "send")
						return
					}
					if block != nil {
						// Wrap block
						mqAccountBlock := &MqAccountBlock{}
						mqAccountBlock.AccountBlock = *block

						var sendBlock *ledger.AccountBlock

						var tokenTypeId *types.TokenTypeId
						if block.IsReceiveBlock() {
							var err error
							sendBlock, err = producer.chain.GetAccountBlockByHash(&block.FromBlockHash)

							if err != nil {
								producer.log.Error("Get send account block failed, error is "+err.Error(), "method", "send")
								return
							}

							if sendBlock != nil {
								tokenTypeId = &sendBlock.TokenId
								// set token id
								mqAccountBlock.Amount = sendBlock.Amount
								mqAccountBlock.TokenId = sendBlock.TokenId
								mqAccountBlock.FromAddress = sendBlock.AccountAddress
								mqAccountBlock.ToAddress = mqAccountBlock.AccountAddress
								mqAccountBlock.SendData = sendBlock.Data
							}
						} else {
							tokenTypeId = &block.TokenId
							mqAccountBlock.FromAddress = mqAccountBlock.AccountAddress

							var err error
							mqAccountBlock.ParsedData, err = producer.getParsedData(block)
							if err != nil {
								producer.log.Error("GetParsedData failed, error is "+err.Error(), "method", "send")
								return
							}

						}

						balance := big.NewInt(0)

						if tokenTypeId != nil {
							vc, newVcErr := vm_context.NewVmContext(producer.chain, nil, &block.Hash, &block.AccountAddress)
							if newVcErr != nil {
								producer.log.Error("NewVmContext failed, error is "+newVcErr.Error(), "method", "send")
								return
							}
							balance = vc.GetBalance(nil, tokenTypeId)
						}

						mqAccountBlock.Balance = balance
						mqAccountBlock.Timestamp = block.Timestamp.Unix()
						blocks = append(blocks, mqAccountBlock)
					}
				}

				if len(blocks) <= 0 {
					producer.hasSend = j
					continue
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
				var blocks []*MqSnapshotBlock
				for _, blockHash := range blockHashList {
					block, err := producer.chain.GetSnapshotBlockByHash(&blockHash)
					if err != nil {
						producer.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "send")
						return
					}
					if block != nil {
						mqSnapshotBlock := &MqSnapshotBlock{}
						mqSnapshotBlock.SnapshotBlock = block
						subLedger, err := producer.chain.GetConfirmSubLedgerBySnapshotBlocks([]*ledger.SnapshotBlock{block})
						if err != nil {
							producer.log.Error("GetConfirmSubLedgerBySnapshotBlocks failed, error is "+err.Error(), "method", "send")
							return
						}

						mqSnapshotBlock.MqSnapshotContent = make(MqSnapshotContent)
						for addr, blocks := range subLedger {
							mqSnapshotBlock.MqSnapshotContent[addr] = &MqSnapshotContentItem{
								Start: &ledger.HashHeight{
									Hash:   blocks[0].Hash,
									Height: blocks[0].Height,
								},
								End: &ledger.HashHeight{
									Hash:   blocks[len(blocks)-1].Hash,
									Height: blocks[len(blocks)-1].Height,
								},
							}
						}

						mqSnapshotBlock.Producer = mqSnapshotBlock.SnapshotBlock.Producer()
						mqSnapshotBlock.Timestamp = block.Timestamp.Unix()
						blocks = append(blocks, mqSnapshotBlock)
					}
				}

				if len(blocks) <= 0 {
					producer.hasSend = j
					continue
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
				producer.hasSend = j
				continue
			}

			msgList = append(msgList, m)

		}

		sendErr := producer.sendMessage(msgList)
		if sendErr != nil {
			producer.log.Error("sendMessage failed, error is "+sendErr.Error(), "method", "send")
			return
		}

		i = j - 1
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

func (producer *Producer) sendMessage(msgList []*message) (err error) {
	//var err error
	for i := 0; i < len(msgList); i++ {
		buf, jsonErr := json.Marshal(msgList[i])
		if jsonErr != nil {
			return jsonErr
		}
		sMsg := &sarama.ProducerMessage{Topic: producer.topic, Value: sarama.StringEncoder(buf)}

		producer.sendWg.Add(1)
		// Simple implementation, may be fix
		go func() {
			defer producer.sendWg.Done()
			producer.kafkaProducer.Input() <- sMsg
			select {
			// success
			case <-producer.kafkaProducer.Successes():
				break

			// error
			case sendError := <-producer.kafkaProducer.Errors():
				producer.log.Error("kafka send failed, error is "+sendError.Error(), "method", "sendMessage")
				err = sendError
			}
		}()
	}
	producer.sendWg.Wait()
	return
}
