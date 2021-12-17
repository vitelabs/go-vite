package producer

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/config"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/common/upgrade"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/ledger/consensus"
	"github.com/vitelabs/go-vite/ledger/pool"
	"github.com/vitelabs/go-vite/ledger/verifier"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/wallet"
)

var log = log15.New("module", "producerTest")

func genConsensus(c chain.Chain, pool pool.BlockPool, t *testing.T) consensus.Consensus {
	cs := consensus.NewConsensus(c, pool)
	err := cs.Init(nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cs.Start()
	return cs
}

func TestSnapshot(t *testing.T) {
	upgrade.CleanupUpgradeBox(t)
	upgrade.InitUpgradeBox(upgrade.NewLatestUpgradeBox())

	tmpDir, _ := ioutil.TempDir("", "")
	c := chain.NewChain(tmpDir, nil, config.MockGenesis())
	c.Init()
	c.Start()

	p1, _ := pool.NewPool(c)
	cs := genConsensus(c, p1, t)
	coinbase, err := wallet.RandomAccount()

	helper.ErrFailf(t, err, "random account err")

	sv := verifier.NewSnapshotVerifier(c, cs)
	av := verifier.NewVerifier2(c, cs)
	p := NewProducer(c, net.Mock(c), coinbase, cs, sv, p1)

	p1.Init(net.Mock(c), sv, av, cs)
	p.Init()

	address := coinbase.Address()
	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", &address, func(e consensus.Event) {
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
	upgrade.CleanupUpgradeBox(t)
	upgrade.InitUpgradeBox(upgrade.NewEmptyUpgradeBox())
	tmpDir, _ := ioutil.TempDir("", "")
	c := chain.NewChain(tmpDir, nil, config.MockGenesis())
	c.Init()

	p1, _ := pool.NewPool(c)
	coinbase, _ := wallet.RandomAccount()
	cs := genConsensus(c, p1, t)
	sv := verifier.NewSnapshotVerifier(c, cs)
	av := verifier.NewVerifier2(c, cs)
	p := NewProducer(c, net.Mock(c), coinbase, cs, sv, p1)

	c.Start()

	p1.Init(net.Mock(c), sv, av, cs)
	p.Init()
	p1.Start()
	p.Start()

	e := consensus.Event{
		Gid:       types.SNAPSHOT_GID,
		Address:   coinbase.Address(),
		Stime:     time.Now(),
		Etime:     time.Now(),
		Timestamp: time.Now(),
	}

	t.Log(c.GetLatestSnapshotBlock().Height)

	p.worker.produceSnapshot(e)
	time.Sleep(2 * time.Second)
	p.Stop()
	t.Log(c.GetLatestSnapshotBlock())
	t.Log(c.GetLatestSnapshotBlock().Height)
}
