package verifier

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
)

var innerChainInstance chain.Chain

func getChainInstance() chain.Chain {
	if innerChainInstance == nil {
		home := common.HomeDir()

		innerChainInstance = chain.NewChain(&config.Config{
			DataDir: filepath.Join(home, "Documents/vite/src/github.com/vitelabs/aaaaaaaa/devdata"),
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

func TestSnapshotBlockVerify(t *testing.T) {
	chainInstance := getChainInstance()

	v := NewSnapshotVerifier(chainInstance, nil)
	head := chainInstance.GetLatestSnapshotBlock()
	t.Log(head)
	for i := uint64(1); i <= head.Height; i++ {
		block, e := chainInstance.GetSnapshotBlockByHeight(i)
		if e == nil {
			if v.VerifyNetSb(block) != nil {
				t.Log(fmt.Sprintf("%+v\n", block))
			}
		}
	}
}
