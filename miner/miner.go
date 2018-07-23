package miner

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/hdops"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

// Package miner implements vite block creation

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	WriteMiningBlock(block *ledger.SnapshotBlock) error
}

type SnapshotBlock struct {
}

type Miner struct {
	types.LifecycleStatus
	chain     SnapshotChainRW
	mining    int32
	coinbase  types.Address // address
	worker    *worker
	committee *consensus.Committee
	mem       *consensus.SubscribeMem
}

func NewMiner(chain SnapshotChainRW, coinbase types.Address) *Miner {
	miner := &Miner{chain: chain, coinbase: coinbase}

	genesisTime := time.Unix(int64(ledger.GetSnapshotGenesisBlock().Timestamp), 0)
	miner.committee = consensus.NewCommittee(genesisTime, 6, int32(len(consensus.DefaultMembers)))
	miner.mem = &consensus.SubscribeMem{Mem: miner.coinbase, Notify: make(chan time.Time)}
	miner.worker = &worker{chain: chain, workChan: &miner.mem.Notify, coinbase: coinbase}
	return miner
}
func (self *Miner) Init() {
	self.PreInit()
	defer self.PostInit()
	self.worker.Init()
	self.committee.Subscribe(self.mem)
	self.committee.Init()
}

func (self *Miner) Start() {
	self.PreStart()
	defer self.PostStart()

	self.worker.Start()
	self.committee.Start()
}

func (self *Miner) Stop() {
	self.PreStop()
	defer self.PostStop()

	self.worker.Stop()
	close(self.mem.Notify)
	self.committee.Stop()
}
