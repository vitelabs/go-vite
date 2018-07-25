package miner

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log"
	"time"
)

// Package miner implements vite block creation

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	WriteMiningBlock(block *ledger.SnapshotBlock) error
}

type DownloaderRegister func(chan<- int) // 0 represent success, not 0 represent failed.

type Miner struct {
	types.LifecycleStatus
	chain                SnapshotChainRW
	mining               int32
	coinbase             types.Address // address
	worker               *worker
	committee            *consensus.Committee
	mem                  *consensus.SubscribeMem
	downloaderRegister   DownloaderRegister
	downloaderRegisterCh chan int
}

func NewMiner(chain SnapshotChainRW, downloaderRegister DownloaderRegister,coinbase types.Address, committee *consensus.Committee) *Miner {
	miner := &Miner{chain: chain, coinbase: coinbase}

	miner.committee = committee
	miner.mem = &consensus.SubscribeMem{Mem: miner.coinbase, Notify: make(chan time.Time)}
	miner.worker = &worker{chain: chain, workChan: miner.mem.Notify, coinbase: coinbase}
	miner.downloaderRegister = downloaderRegister
	miner.downloaderRegisterCh = make(chan int)
	return miner
}
func (self *Miner) Init() {
	self.PreInit()
	defer self.PostInit()
	self.worker.Init()
	self.downloaderRegister(self.downloaderRegisterCh)
	go func() {
		select {
		// Handle ChainHeadEvent
		case event := <-self.downloaderRegisterCh:
			if event == 0 {
				log.Info("downloader success.")
				self.committee.Subscribe(self.mem)
			} else {
				log.Error("downloader error.")
			}

		}
	}()
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
