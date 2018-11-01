package chain

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"path/filepath"
)

var innerChainInstance Chain

func getChainInstance() Chain {
	if innerChainInstance == nil {
		//home := common.HomeDir()

		innerChainInstance = NewChain(&config.Config{
			//DataDir: filepath.Join(common.HomeDir(), "govite_testdata"),

			DataDir: filepath.Join(common.HomeDir(), "Library/GVite/devdata"),
			//Chain: &config.Chain{
			//	KafkaProducers: []*config.KafkaProducer{{
			//		Topic:      "test003",
			//		BrokerList: []string{"ckafka-r3rbhht9.ap-guangzhou.ckafka.tencentcloudmq.com:6061"},
			//	}},
			//},
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}

	return innerChainInstance
}
