package pool

import (
	"encoding/base64"
	"testing"

	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"

	"time"

	"path/filepath"

	"fmt"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
)

var innerChainInstance ch.Chain

func getChainInstance() ch.Chain {
	if innerChainInstance == nil {
		//home := common.HomeDir()

		innerChainInstance = ch.NewChain(&config.Config{
			//DataDir: filepath.Join(common.HomeDir(), "govite_testdata"),

			DataDir: filepath.Join(common.HomeDir(), "viteisbest"),
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

func TestChain(t *testing.T) {
	c := getChainInstance()
	block, e := c.GetSnapshotBlockByHeight(3574)
	if e != nil {
		panic(e)
	}
	fmt.Println(block.Hash)

	for k, v := range block.SnapshotContent {
		fmt.Println(k, v.Hash, v.Height)
	}
}

func TestChain2(t *testing.T) {
	c := getChainInstance()
	hashes, e := types.HexToHash("8ae598902a4f30ff2390914b8a2b0172fdbc2400348b73231462380e65323b0d")
	if e != nil {
		panic(e)
	}
	block, e := c.GetAccountBlockByHash(&hashes)
	if e != nil {
		panic(e)
	}
	fmt.Println(block.Hash, block.Height)

	addr, e := types.HexToAddress("vite_d6851aaf8966f4550bde3d64582f7da9c6ab8b18a289823b95")
	if e != nil {
		panic(e)
	}
	head, e := c.GetLatestAccountBlock(&addr)
	if e != nil {
		panic(e)
	}
	fmt.Println(head.Height, head.Hash)

}
func TestNewPool(t *testing.T) {
	c := ch.NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
	})
	p := NewPool(c)
	w := wallet.New(nil)

	av := verifier.NewAccountVerifier(c, nil)

	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)

	c.Init()
	c.Start()
	p.Init(&MockSyncer{}, w, sv, av)

	p.Start()

	block := c.GetLatestSnapshotBlock()
	t.Log(block.Height, block.Hash, block.PrevHash, block.Producer())
	for k, v := range block.SnapshotContent {
		t.Log(k.String(), v.Hash, v.Height)
	}
}

func TestPool(t *testing.T) {
	c := ch.NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
	})
	p := NewPool(c)
	w := wallet.New(nil)

	av := verifier.NewAccountVerifier(c, nil)

	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)

	c.Init()
	c.Start()
	p.Init(&MockSyncer{}, w, sv, av)

	p.Start()

	block := c.GetLatestSnapshotBlock()
	t.Log(block.Height, block.Hash, block.PrevHash, block.Producer())
	for k, v := range block.SnapshotContent {
		t.Log(k.String(), v.Hash, v.Height)
	}
}

func TestPool_Lock(t *testing.T) {
	c := ch.NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
	})
	p := NewPool(c)
	w := wallet.New(nil)

	av := verifier.NewAccountVerifier(c, nil)

	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)

	c.Init()
	c.Start()
	p.Init(&MockSyncer{}, w, sv, av)

	p.Start()

	block := c.GetLatestSnapshotBlock()
	t.Log(block.Height, block.Hash, block.PrevHash, block.Producer())
	for k, v := range block.SnapshotContent {
		t.Log(k.String(), v.Hash, v.Height)
	}

	p.Lock()

	a := 0
	go func() {
		a++
		p.RLock()
		a++
		defer p.RUnLock()
	}()

	for a < 1 {
		time.Sleep(2 * time.Second)
	}

	time.Sleep(2 * time.Second)
	if a == 2 {
		t.Error(a)
	}
	p.UnLock()

}

func TestAddress(t *testing.T) {
	bytes, e := base64.StdEncoding.DecodeString("jbqarcyPTcv32/E+CHrdXQKjQNQlgOIOOS0l7cOp47A=")
	if e != nil {
		t.Fatal(e)
	}
	address := types.PubkeyToAddress(bytes)

	t.Log(address)
}
