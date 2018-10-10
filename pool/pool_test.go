package pool

import (
	"testing"

	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"

	"github.com/vitelabs/go-vite/common"
)

func TestNewPool(t *testing.T) {
	innerChainInstance := ch.NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
		//Chain: &config.Chain{
		//	KafkaProducers: []*config.KafkaProducer{{
		//		Topic:      "test",
		//		BrokerList: []string{"abc", "def"},
		//	}},
		//},
	})
	innerChainInstance.Init()
	innerChainInstance.Start()
	newPool := NewPool(innerChainInstance)
	newPool.Init(&MockSyncer{})
	newPool.Start()

	make(chan int) <- 1
}
