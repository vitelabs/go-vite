package chain

import (
	"path/filepath"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
)

var innerChainInstance Chain

func getChainInstance() Chain {
	if innerChainInstance == nil {
		//home := common.HomeDir()

		innerChainInstance = NewChain(&config.Config{
			DataDir: filepath.Join(common.HomeDir(), "Library/GVite/devdata"),

			//DataDir: filepath.Join(common.HomeDir(), "Library/GVite/devdata"),
			//Chain: &config.Chain{
			//	KafkaProducers: []*config.KafkaProducer{{
			//		Topic:      "test003",
			//		BrokerList: []string{"ckafka-r3rbhht9.ap-guangzhou.ckafka.tencentcloudmq.com:6061"},
			//	}},
			//},
			Chain: &config.Chain{GenesisFile: "/Users/jie/Documents/vite/src/github.com/vitelabs/genesis.json"},
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}

	return innerChainInstance
}
