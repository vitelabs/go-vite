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

func (*testSubscriber) SyncState() net.SyncState {
	return net.SyncDone
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
		fn(net.SyncDone)
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

	addr, err := types.HexToAddress("vite_91dc0c38d104c7915d3a6c4381a40c360edd871c34ac255bb2")
	if err != nil {
		panic(err)
	}
	coinbase := &AddressContext{
		EntryPath: "/Users/jie/viteisbest/wallet2/vite_91dc0c38d104c7915d3a6c4381a40c360edd871c34ac255bb2",
		Address:   addr,
		Index:     0,
	}

	sv := verifier.NewSnapshotVerifier(c, cs)
	w := wallet.New(nil)
	av := verifier.NewAccountVerifier(c, cs)
	p1, _ := pool.NewPool(c)
	p := NewProducer(c, &testSubscriber{}, coinbase, cs, sv, w, p1)

	p1.Init(&pool.MockSyncer{}, w, sv, av)
	p.Init()

	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", &coinbase.Address, func(e consensus.Event) {
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
	addr, err := types.HexToAddress("vite_91dc0c38d104c7915d3a6c4381a40c360edd871c34ac255bb2")
	if err != nil {
		panic(err)
	}
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})

	coinbase := &AddressContext{
		EntryPath: "/Users/jie/viteisbest/wallet2/vite_91dc0c38d104c7915d3a6c4381a40c360edd871c34ac255bb2",
		Address:   addr,
		Index:     0,
	}
	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)
	w := wallet.New(nil)
	av := verifier.NewAccountVerifier(c, cs)
	p1, _ := pool.NewPool(c)
	p := NewProducer(c, &testSubscriber{}, coinbase, cs, sv, w, p1)

	c.Init()
	c.Start()

	p1.Init(&pool.MockSyncer{}, w, sv, av)
	p.Init()
	p1.Start()
	p.Start()

	e := consensus.Event{
		Gid:            types.SNAPSHOT_GID,
		Address:        coinbase.Address,
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
