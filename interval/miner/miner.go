package miner

import (
	"sync/atomic"
	"time"

	"github.com/asaskevich/EventBus"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/consensus"
)

// Package miner implements vite block creation

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	MiningSnapshotBlock(address common.Address, timestamp int64) error
}

type DownloaderRegister func(chan<- int) // 0 represent success, not 0 represent failed.

/**

0->1->2->3->4->5->6->7->8
		 ^|_______\
*/
// 0:origin 1: initing 2:inited 3:starting 4:started 5:stopping 6:stopped 7:destroying 8:destroyed
type MinerLifecycle struct {
	common.LifecycleStatus
}

func (l *MinerLifecycle) PreDestroy() bool {
	return atomic.CompareAndSwapInt32(&l.S, 6, 7)
}
func (l *MinerLifecycle) PostDestroy() bool {
	return atomic.CompareAndSwapInt32(&l.S, 7, 8)
}

func (l *MinerLifecycle) PreStart() bool {
	return atomic.CompareAndSwapInt32(&l.S, 2, 3) || atomic.CompareAndSwapInt32(&l.S, 6, 3)
}
func (l *MinerLifecycle) PostStart() bool {
	return atomic.CompareAndSwapInt32(&l.S, 3, 4)
}

type Miner interface {
	Init()
	Start()
	Stop()
}

type miner struct {
	MinerLifecycle
	chain      SnapshotChainRW
	mining     int32
	coinbase   common.Address // address
	worker     *worker
	consensus  consensus.Consensus
	mem        *consensus.SubscribeMem
	bus        EventBus.Bus
	syncStatus face.SyncStatus
}

func NewMiner(chain SnapshotChainRW, syncStatus face.SyncStatus, bus EventBus.Bus, coinbase common.Address, con consensus.Consensus) Miner {
	miner := &miner{chain: chain, coinbase: coinbase}

	miner.consensus = con
	miner.mem = &consensus.SubscribeMem{Mem: miner.coinbase, Notify: make(chan time.Time)}
	miner.worker = &worker{chain: chain, workChan: miner.mem.Notify, coinbase: coinbase}
	miner.bus = bus
	miner.syncStatus = syncStatus

	return miner
}
func (m *miner) Init() {
	m.PreInit()
	defer m.PostInit()
	m.worker.Init()
	dwlDownFn := func() {
		log.Info("sync success.")
		m.consensus.Subscribe(m.mem)
	}
	m.bus.SubscribeOnce(common.DwlDone, dwlDownFn)
}

func (m *miner) Start() {
	m.PreStart()
	defer m.PostStart()

	if m.syncStatus.Done() {
		m.consensus.Subscribe(m.mem)
	}
	m.worker.Start()
}

func (m *miner) Stop() {
	m.PreStop()
	defer m.PostStop()

	m.worker.Stop()
	m.consensus.Subscribe(nil)
}

func (m *miner) Destroy() {
	m.PreDestroy()
	defer m.PostDestroy()
}
