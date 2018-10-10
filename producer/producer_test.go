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
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/wallet"
)

type testConsensus struct {
}

func (*testConsensus) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(consensus.Event)) {
}

func (*testConsensus) UnSubscribe(gid types.Gid, id string) {
}

var log = log15.New("module", "producerTest")

var accountPrivKeyStr string

func init() {
	flag.StringVar(&accountPrivKeyStr, "k", "", "")
	flag.Parse()
	fmt.Println(accountPrivKeyStr)

}

func genConsensus(c chain.Chain, t *testing.T) consensus.Consensus {

	genesis := chain.GenesisSnapshotBlock
	cs := consensus.NewConsensus(*genesis.Timestamp, c)
	err := cs.Init()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	cs.Start()
	return cs
}

type testSubscriber struct {
}

func (*testSubscriber) SubscribeAccountBlock(fn net.AccountblockCallback) (subId int) {
	panic("implement me")
}

func (*testSubscriber) UnsubscribeAccountBlock(subId int) {
	panic("implement me")
}

func (*testSubscriber) SubscribeSnapshotBlock(fn net.SnapshotBlockCallback) (subId int) {
	panic("implement me")
}

func (*testSubscriber) UnsubscribeSnapshotBlock(subId int) {
	panic("implement me")
}

func (*testSubscriber) SubscribeSyncStatus(fn net.SyncStateCallback) (subId int) {
	go func() {
		time.Sleep(2 * time.Second)
		fn(net.Syncdone)
	}()
	return 0
}

func (*testSubscriber) UnsubscribeSyncStatus(subId int) {

}

func TestSnapshot(t *testing.T) {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	cs := genConsensus(c, t)

	accountPrivKey, _ := ed25519.HexToPrivateKey(accountPrivKeyStr)
	accountPubKey := accountPrivKey.PubByte()
	coinbase := types.PubkeyToAddress(accountPubKey)

	sv := verifier.NewSnapshotVerifier(c, cs)
	w := wallet.New(nil)
	av := verifier.NewAccountVerifier(c, cs)
	p1 := pool.NewPool(c)
	p := NewProducer(c, &testSubscriber{}, coinbase, cs, sv, w, p1)

	w.KeystoreManager.ImportPriv(accountPrivKeyStr, "123456")
	w.KeystoreManager.Lock(coinbase)
	w.KeystoreManager.Unlock(coinbase, "123456", 0)
	log.Info("unlock address", "address", coinbase.String())
	locked := w.KeystoreManager.IsUnLocked(coinbase)
	if !locked {
		t.Error("unlock failed.")
		return
	}

	p1.Init(&pool.MockSyncer{}, w, sv, av)
	p.Init()

	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", &coinbase, func(e consensus.Event) {
		log.Info("snapshot", "e", e)
	})
	p1.Start()
	p.Start()

	go func() {
		for {
			head := c.GetLatestSnapshotBlock()
			log.Info("head info", "hash", head.Hash.String(), "height", head.Height)
			time.Sleep(1 * time.Second)
		}
	}()

	make(chan int) <- 1
}

func TestProducer_Init(t *testing.T) {
	accountPrivKey, _ := ed25519.HexToPrivateKey(accountPrivKeyStr)
	accountPubKey := accountPrivKey.PubByte()
	coinbase := types.PubkeyToAddress(accountPubKey)

	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})

	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)
	w := wallet.New(nil)
	av := verifier.NewAccountVerifier(c, cs)
	p1 := pool.NewPool(c)
	p := NewProducer(c, &testSubscriber{}, coinbase, cs, sv, w, p1)

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
