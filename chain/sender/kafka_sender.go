package sender

import (
	"github.com/Shopify/sarama"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"sync"
)

const (
	STOPPED = iota
	RUNNING
)

type kafkaMessage struct {
	MsgType string `json:"type"`
	Data    string `json:"data"`
}

type KafkaSender struct {
	producer sarama.AsyncProducer

	wg          sync.WaitGroup
	termination chan int

	status     int
	statusLock sync.Mutex

	log log15.Logger
}

func NewKafkaSender(chain Chain, addr []string, topic string, dirname string) (*KafkaSender, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	producer, err := sarama.NewAsyncProducer(addr, config)
	if err != nil {
		return nil, err
	}

	// create directory
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		os.Mkdir(dirname, 0755)
	}

	sender := &KafkaSender{
		producer: producer,
		log:      log15.New("module", "chain/sender"),
	}

	return sender, nil
}

func (sender *KafkaSender) Start() {
	sender.statusLock.Lock()
	defer sender.statusLock.Unlock()
	if sender.status == RUNNING {
		return
	}

	sender.status = RUNNING
	sender.termination = make(chan int)

	sender.wg.Add(1)
	go func() {
		defer sender.wg.Done()
		for {
			select {
			case sender.termination:
				return
			default:

			}
		}
	}()
}

func (sender *KafkaSender) Stop() {
	sender.statusLock.Lock()
	defer sender.statusLock.Unlock()
	if sender.status == STOPPED {
		return
	}

	sender.termination <- 1
	sender.wg.Wait()

	sender.status = STOPPED
}
