package consensus

import (
	"testing"

	"time"

	"fmt"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
)

var log = log15.New("module", "consensusTest")

func TestConsensus(t *testing.T) {

	ch := make(chan string)

	cs := genConsensus(t)
	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", nil, func(e Event) {
		ch <- fmt.Sprintf("snapshot: %s, %v", e.Address.String(), e)
	})

	cs.Subscribe(types.DELEGATE_GID, "contract_mock", nil, func(e Event) {
		ch <- fmt.Sprintf("account: %s, %v", e.Address.String(), e)
	})

	for {
		msg := <-ch
		log.Info(msg)

	}

}

func TestCommittee_ReadVoteMapByTime(t *testing.T) {
	cs := genConsensus(t)
	now := time.Now()
	u, e := cs.VoteTimeToIndex(types.SNAPSHOT_GID, now)
	if e != nil {
		panic(e)
	}

	details, _, err := cs.ReadVoteMapByTime(types.SNAPSHOT_GID, u)
	if err != nil {
		panic(err)
	}
	for k, v := range details {
		t.Log(k, v.addr, v.name)
	}
}

func TestCommittee_ReadByTime(t *testing.T) {
	cs := genConsensus(t)
	now := time.Now()
	contractR, _, err := cs.ReadByTime(types.DELEGATE_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range contractR {
		t.Log(types.DELEGATE_GID, k, v, err)
	}
	snapshotR, _, err := cs.ReadByTime(types.SNAPSHOT_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range snapshotR {
		t.Log(types.SNAPSHOT_GID, k, v, err)
	}

	if len(contractR)*3 != len(snapshotR) {
		t.Error("len error.")
	}

	contractMap := make(map[types.Address]bool)
	for _, v := range contractR {
		contractMap[v.Address] = true
	}

	for _, v := range snapshotR {
		if contractMap[v.Address] != true {
			t.Error("address err", v.Address.String())
		}
	}
}

func genConsensus(t *testing.T) *committee {
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

func TestChainBlock(t *testing.T) {
	c := genConsensus(t)

	chn := c.rw.rw.(chain.Chain)

	headHeight := chn.GetLatestSnapshotBlock().Height
	log.Info("snapshot head height", "height", headHeight)

	for i := uint64(1); i <= headHeight; i++ {
		block, err := chn.GetSnapshotBlockByHeight(i)
		if err != nil {
			t.Error(err)
		}
		b, err := c.VerifySnapshotProducer(block)
		if !b {
			t.Error("snapshot block verify fail.", "block", block, "err", err)
		}
	}
}
