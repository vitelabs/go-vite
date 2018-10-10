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

var log = log15.New("consensusTest")

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

func TestCommittee_ReadByTime(t *testing.T) {
	cs := genConsensus(t)
	now := time.Now()
	contractR, err := cs.ReadByTime(types.DELEGATE_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range contractR {
		t.Log(types.DELEGATE_GID, k, v, err)
	}
	snapshotR, err := cs.ReadByTime(types.SNAPSHOT_GID, now)

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
