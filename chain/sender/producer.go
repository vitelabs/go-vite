package sender

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"sync"
)

const (
	STOPPED = iota
	RUNNING
)

type Producer struct {
	brokerList []string
	topic      string
	version    string

	hasSend     uint64
	termination chan int

	status     int
	statusLock sync.Mutex
	log        log15.Logger

	wg sync.WaitGroup
}

func NewProducer() *Producer {
	return &Producer{}
}
func (producer *Producer) Deserialize(buffer []byte) error {
	pb := &vitepb.ProducerUnit{}
	if err := proto.Unmarshal(buffer, pb); err != nil {
		return err
	}

	producer.topic = pb.Topic
	producer.brokerList = pb.BrokerList
	producer.version = pb.Version
	producer.hasSend = pb.HasSend
	return nil
}

func (producer *Producer) Start() {
	producer.statusLock.Lock()
	defer producer.statusLock.Unlock()
	if producer.status == RUNNING {
		return
	}

	producer.status = RUNNING
	producer.termination = make(chan int)

	go func() {
		defer producer.wg.Done()
		for {
			select {
			case producer.termination:
				return
			default:

			}
		}
	}()
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
