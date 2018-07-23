package miner

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

// Package miner implements vite block creation

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	WriteMiningBlock(block *ledger.SnapshotBlock) error
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

func NewMiner(chain SnapshotChainRW, coinbase types.Address, committee *consensus.Committee) *Miner {
	miner := &Miner{chain: chain, coinbase: coinbase}

	miner.committee = committee
	miner.mem = &consensus.SubscribeMem{Mem: miner.coinbase, Notify: make(chan time.Time)}
	miner.worker = &worker{chain: chain, workChan: &miner.mem.Notify, coinbase: coinbase}
	return miner
}
func (self *Miner) Init() {
	self.PreInit()
	defer self.PostInit()
	self.worker.Init()
	self.committee.Subscribe(self.mem)
}

func (self *Miner) Start() {
	self.PreStart()
	defer self.PostStart()

	self.worker.Start()
}

func (self *Miner) Stop() {
	self.PreStop()
	defer self.PostStop()

	self.worker.Stop()
	close(self.mem.Notify)
}
