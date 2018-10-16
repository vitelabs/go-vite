package chain

import (
	"path/filepath"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
)

var innerChainInstance Chain

func getChainInstance() Chain {
	if innerChainInstance == nil {
		home := common.HomeDir()

		innerChainInstance = NewChain(&config.Config{
			DataDir: filepath.Join(home, "viteisbest"),
			//Chain: &config.Chain{
			//	KafkaProducers: []*config.KafkaProducer{{
			//		Topic:      "test",
			//		BrokerList: []string{"abc", "def"},
			//	}},
			//},
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}

	return innerChainInstance
}
