package chain

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"os"
	"path/filepath"
)

var innerChainInstance Chain

func getChainInstance() Chain {
	if innerChainInstance == nil {
		//home := common.HomeDir()
		dbFile := common.HomeDir()
		os.RemoveAll(filepath.Join(dbFile, "ledger"))

		innerChainInstance = NewChain(&config.Config{
			DataDir: dbFile,
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
