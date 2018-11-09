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

func getChainInstance(path string) chain.Chain {
	if path == "" {
		path = "Documents/vite/src/github.com/vitelabs/aaaaaaaa/devdata"
	}
	if innerChainInstance == nil {
		home := common.HomeDir()

		innerChainInstance = chain.NewChain(&config.Config{
			DataDir: filepath.Join(home, path),
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
	chainInstance := getChainInstance("")

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

func TestVerifyGenesis(t *testing.T) {
	c := getChainInstance("")
	block := c.GetGenesisSnapshotBlock()
	snapshotBlock, _ := c.GetSnapshotBlockByHeight(1)
	if block.Hash != snapshotBlock.Hash {
		t.Error("snapshot block error.", snapshotBlock, block)
	}
	t.Log(c.GetLatestSnapshotBlock())
}
