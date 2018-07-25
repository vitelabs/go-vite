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
	Ch  chan<- int
}

func (SnapshotRW) WriteMiningBlock(block *ledger.SnapshotBlock) error {
	println(block.Producer.String() + ":" + time.Unix(int64(block.Timestamp), 0).Format(time.StampMilli) + ":" + strconv.Itoa(int(block.Timestamp)))
	return nil
}

func (self *SnapshotRW) funcDownloaderRegister(ch chan<- int) {
	self.Ch = ch
}

func TestNewMiner(t *testing.T) {
	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	committee := consensus.NewCommittee(genesisTime, 1, int32(len(consensus.DefaultMembers)))

	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	rw := &SnapshotRW{}
	miner := NewMiner(rw, rw.funcDownloaderRegister, coinbase, committee)

	committee.Init()
	miner.Init()
	committee.Start()
	miner.Start()
	var c chan int = make(chan int)
	select {
	case c <- 0:
	case <-time.After(5 * time.Second):
		println("timeout and downloader finish.")
		//miner.downloaderRegisterCh <- 0
		rw.Ch <-0
		println("-----------timeout")
	}
	c <- 0
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
