package producer

import (
	"testing"

	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
)

type testConsensus struct {
}

func (*testConsensus) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(consensus.Event)) {
}

func (*testConsensus) UnSubscribe(gid types.Gid, id string) {
}

func TestProducer_Init(t *testing.T) {
	coinbase := common.MockAddress(1)
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})

	cs := &consensus.MockConsensus{}
	v := verifier.NewSnapshotVerifier(c, cs)
	w := wallet.New(nil)
	p := NewProducer(c, nil, coinbase, cs, v, w)

	c.Init()

	p.Init()
	c.Start()
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
	p.Stop()
	t.Log(c.GetLatestSnapshotBlock())
	t.Log(c.GetLatestSnapshotBlock().Height)
}
