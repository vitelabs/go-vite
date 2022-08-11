package producer

import (
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/fileutils"
	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
	"github.com/vitelabs/go-vite/v2/ledger/pool"
	"github.com/vitelabs/go-vite/v2/ledger/verifier"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/net"
	"github.com/vitelabs/go-vite/v2/wallet"
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
	upgrade.CleanupUpgradeBox()
	upgrade.InitUpgradeBox(upgrade.NewLatestUpgradeBox())

	tmpDir := fileutils.CreateTempDir()
	c := chain.NewChain(tmpDir, nil, config.MockGenesis())
	c.Init()
	c.Start()

	p1, _ := pool.NewPool(c)
	cs := genConsensus(c, p1, t)
	coinbase, err := wallet.RandomAccount()

	helper.ErrFailf(t, err, "random account err")

	// sv := verifier.NewSnapshotVerifier(c, cs)
	av := verifier.NewVerifier(c).Init(cs, cs.SBPReader(), nil)
	p := NewProducer(c, net.Mock(c), coinbase, cs, p1)

	p1.Init(net.Mock(c), av, cs.SBPReader())
	p.Init()

	address := coinbase.Address()
	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", &address, func(e consensus.Event) {
		log.Info("snapshot", "e", e)
	})
	p1.Start()
	p.Start()

	delay := 1 * time.Second
	ellapsed := time.Duration(0)
	ch := make(chan int)
	var once sync.Once

	go func() {
		for {
			head := c.GetLatestSnapshotBlock()
			log.Info("head info", "hash", head.Hash.String(), "height", head.Height, "ellapsed", ellapsed)
			time.Sleep(delay)
			ellapsed += delay
			// Increase max. duration as needed
			if ellapsed >= time.Duration(3)*time.Second {
				once.Do(func() {
					close(ch)
				})
			}
		}
	}()

	<-ch
}

func TestProducer_Init(t *testing.T) {
	upgrade.CleanupUpgradeBox()
	upgrade.InitUpgradeBox(upgrade.NewEmptyUpgradeBox())
	tmpDir := fileutils.CreateTempDir()
	c := chain.NewChain(tmpDir, nil, config.MockGenesis())
	c.Init()

	p1, _ := pool.NewPool(c)
	coinbase, _ := wallet.RandomAccount()
	cs := genConsensus(c, p1, t)
	// sv := verifier.NewSnapshotVerifier(c, cs)
	av := verifier.NewVerifier(c).Init(cs, cs.SBPReader(), nil)
	p := NewProducer(c, net.Mock(c), coinbase, cs, p1)

	c.Start()

	p1.Init(net.Mock(c), av, cs.SBPReader())
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
