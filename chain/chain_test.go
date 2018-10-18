package chain

import (
	"path/filepath"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"os"
)

var innerChainInstance Chain

func getChainInstance() Chain {
	if innerChainInstance == nil {
		dbFile := filepath.Join(common.GoViteTestDataDir(), "ledger")
		os.RemoveAll(dbFile)

		innerChainInstance = NewChain(&config.Config{
			DataDir: filepath.Join(dbFile),
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
