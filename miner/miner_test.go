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
	Ch chan<- int
}

func (SnapshotRW) WriteMiningBlock(block *ledger.SnapshotBlock) error {
	println(block.Producer.String() + ":" + time.Unix(int64(block.Timestamp), 0).Format(time.StampMilli) + ":" + strconv.Itoa(int(block.Timestamp)))
	return nil
}

func (self *SnapshotRW) funcDownloaderRegister(ch chan<- int) {
	self.Ch = ch
}

func (self *SnapshotRW) funcDownloaderRegisterAuto(ch chan<- int) {
	self.Ch = ch
	go func() {
		time.Sleep(1 * time.Second)
		self.Ch <- 0
	}()
}

func genMiner(committee *consensus.Committee) (*Miner, *SnapshotRW) {
	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	rw := &SnapshotRW{}
	miner := NewMiner(rw, rw.funcDownloaderRegister, coinbase, committee)
	return miner, rw
}

func genMinerAuto(committee *consensus.Committee) (*Miner, *SnapshotRW) {
	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	rw := &SnapshotRW{}
	miner := NewMiner(rw, rw.funcDownloaderRegisterAuto, coinbase, committee)
	return miner, rw
}

func genCommitee() *consensus.Committee {
	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	committee := consensus.NewCommittee(genesisTime, 1, int32(len(consensus.DefaultMembers)))
	return committee
}

func TestNewMiner(t *testing.T) {
	committee := genCommitee()
	miner, rw := genMiner(committee)

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
		rw.Ch <- 0
		println("-----------timeout")
	}
	c <- 0
}
func TestVerifier(t *testing.T) {
	committee := genCommitee()

	coinbase, _ := types.HexToAddress("vite_2ad1b8f936f015fc80a2a5857dffb84b39f7675ab69ae31fc8")
	verify, _ := committee.Verify(SnapshotRW{}, &ledger.SnapshotBlock{Producer: &coinbase, Timestamp: uint64(1532504321)})
	println(verify)
	verify2, _ := committee.Verify(SnapshotRW{}, &ledger.SnapshotBlock{Producer: &coinbase, Timestamp: uint64(1532504320)})
	println(verify2)

}

func TestChan(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := make(chan int)

	go func() {
		select {
		// Handle ChainHeadEvent
		case event := <-ch1:
			println(event)
		case e2, ok := <-ch2: // ok代表channel是否正常使用, 如果ok==false, 说明channel已经关闭
			println(e2)
			println(ok)
			println("------")
		}

		ch3 <- 99

	}()

	time.Sleep(1 * time.Second)

	//ch2 <-10
	close(ch2)

	i := <-ch3

	println(i)
}

func TestLifecycle(t *testing.T) {
	commitee := genCommitee()
	miner, _ := genMinerAuto(commitee)

	commitee.Init()
	miner.Init()

	commitee.Start()

	miner.Start()
	var c chan int = make(chan int)

	//
	time.Sleep(30 * time.Second)
	println("miner stop.")
	miner.Stop()
	time.Sleep(1 * time.Second)

	println("miner start.")
	miner.Start()

	time.Sleep(20 * time.Second)
	println("miner stop.")
	miner.Stop()
	time.Sleep(1 * time.Second)

	c <- 0
}
