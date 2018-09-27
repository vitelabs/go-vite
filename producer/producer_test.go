package producer

import (
	"testing"

	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
)

type testConsensus struct {
}

func (*testConsensus) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(consensus.Event)) {
}

func (*testConsensus) UnSubscribe(gid types.Gid, id string) {
}

func TestProducer(t *testing.T) {
	coinbase := common.MockAddress(1)
	p := NewProducer(nil, nil, coinbase, &testConsensus{})
	p.Init()
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
	p.worker.produceSnapshot(e)
	p.Stop()
}
