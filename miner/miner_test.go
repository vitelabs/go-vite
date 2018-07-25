package miner

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"strconv"
	"testing"
	"time"
)

type SnapshotRW struct {
}

func (SnapshotRW) WriteMiningBlock(block *ledger.SnapshotBlock) error {
	println(block.Producer.String() + ":" + time.Unix(int64(block.Timestamp), 0).Format(time.StampMilli) + ":" + strconv.Itoa(int(block.Timestamp)))
	return nil
}

func TestNewMiner(t *testing.T) {
	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	committee := consensus.NewCommittee(genesisTime, 6, int32(len(consensus.DefaultMembers)))

	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	miner := NewMiner(SnapshotRW{}, coinbase, committee)
	committee.Init()
	miner.Init()
	committee.Start()
	miner.Start()
	var c chan int = make(chan int)
	c <- 6
}

func TestVerifier(t *testing.T) {
	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	committee := consensus.NewCommittee(genesisTime, 6, int32(len(consensus.DefaultMembers)))

	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	verify, _ := committee.Verify(SnapshotRW{}, &ledger.SnapshotBlock{Producer: &coinbase, Timestamp: uint64(1532504321)})
	println(verify)
	verify2, _ := committee.Verify(SnapshotRW{}, &ledger.SnapshotBlock{Producer: &coinbase, Timestamp: uint64(1532504320)})
	println(verify2)

}
