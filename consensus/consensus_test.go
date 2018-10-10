package consensus

import (
	"testing"

	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
)

func TestConsensus(t *testing.T) {
	cs := genConsensus(t)
	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", nil, func(e Event) {
		t.Log("snapshot", e.Address, e)
	})

	cs.Subscribe(types.DELEGATE_GID, "contract_mock", nil, func(e Event) {
		t.Log("account", e.Address, e)
	})

}

func TestCommittee_ReadByTime(t *testing.T) {
	cs := genConsensus(t)
	now := time.Now()
	es, err := cs.ReadByTime(types.DELEGATE_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range es {
		t.Log(k, v, err)
	}
	es, err = cs.ReadByTime(types.SNAPSHOT_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range es {
		t.Log(k, v, err)
	}

}

func genConsensus(t *testing.T) Consensus {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	genesis := chain.GenesisSnapshotBlock
	cs := NewConsensus(*genesis.Timestamp, c)
	err := cs.Init()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	cs.Start()
	return cs
}
