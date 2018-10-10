package producer

import (
	"testing"

	"time"

	"flag"
	"fmt"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
)

type testConsensus struct {
}

func (*testConsensus) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(consensus.Event)) {
}

func (*testConsensus) UnSubscribe(gid types.Gid, id string) {
}

var accountPrivKeyStr string

func init() {
	flag.StringVar(&accountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(accountPrivKeyStr)

}

func TestProducer_Init(t *testing.T) {

	accountPrivKey, _ := ed25519.HexToPrivateKey(accountPrivKeyStr)
	accountPubKey := accountPrivKey.PubByte()
	coinbase := types.PubkeyToAddress(accountPubKey)

	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})

	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)
	w := wallet.New(nil)
	av := verifier.NewAccountVerifier(c, cs, w.KeystoreManager)
	p1 := pool.NewPool(c)
	p := NewProducer(c, nil, coinbase, cs, sv, w, p1)

	w.KeystoreManager.ImportPriv(accountPrivKeyStr, "123456")
	w.KeystoreManager.Lock(coinbase)
	w.KeystoreManager.Unlock(coinbase, "123456", 0)

	c.Init()
	c.Start()

	p1.Init(&pool.MockSyncer{}, w, sv, av)
	p.Init()
	p1.Start()
	p.Start()

	e := consensus.Event{
		Gid:            types.SNAPSHOT_GID,
		Address:        coinbase,
		Stime:          time.Now(),
		Etime:          time.Now(),
		Timestamp:      time.Now(),
		SnapshotHeight: 1,
		SnapshotHash:   types.Hash{},
	}

	t.Log(c.GetLatestSnapshotBlock().Height)

	p.worker.produceSnapshot(e)
	time.Sleep(2 * time.Second)
	p.Stop()
	t.Log(c.GetLatestSnapshotBlock())
	t.Log(c.GetLatestSnapshotBlock().Height)
}
