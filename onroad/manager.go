package onroad

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad/model"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
)

var (
	slog           = log15.New("module", "onroad")
	ErrNotSyncDone = errors.New("network synchronization is not complete")
)

type Manager struct {
	keystoreManager *keystore.Manager

	pool     Pool
	net      Net
	chain    chain.Chain
	producer Producer
	wallet   *wallet.Manager

	uAccess          *model.UAccess
	onroadBlocksPool *model.OnroadBlocksPool

	autoReceiveWorkers map[types.Address]*AutoReceiveWorker
	contractWorkers    map[types.Gid]*ContractWorker

	unlockLid       int
	netStateLid     int
	writeOnRoadLid  uint64
	deleteOnRoadLid uint64
	writeSuccLid    uint64
	deleteSuccLid   uint64

	lastProducerAccEvent *producerevent.AccountStartEvent

	log log15.Logger
}

func NewManager(net Net, pool Pool, producer Producer, wallet *wallet.Manager) *Manager {
	m := &Manager{
		pool:               pool,
		net:                net,
		producer:           producer,
		wallet:             wallet,
		keystoreManager:    wallet.KeystoreManager,
		autoReceiveWorkers: make(map[types.Address]*AutoReceiveWorker),
		contractWorkers:    make(map[types.Gid]*ContractWorker),
		log:                slog.New("w", "manager"),
	}
	m.uAccess = model.NewUAccess()
	m.onroadBlocksPool = model.NewOnroadBlocksPool(m.uAccess)
	return m
}

func (manager *Manager) Init(chain chain.Chain) {
	manager.uAccess.Init(chain)
	manager.chain = chain
}

func (manager *Manager) Start() {
	manager.netStateLid = manager.Net().SubscribeSyncStatus(manager.netStateChangedFunc)
	manager.unlockLid = manager.keystoreManager.AddLockEventListener(manager.addressLockStateChangeFunc)
	if manager.producer != nil {
		manager.producer.SetAccountEventFunc(manager.producerStartEventFunc)
	}

	manager.writeSuccLid = manager.Chain().RegisterInsertAccountBlocksSuccess(manager.onroadBlocksPool.WriteOnroadSuccess)
	manager.writeOnRoadLid = manager.Chain().RegisterInsertAccountBlocks(manager.onroadBlocksPool.WriteOnroad)

	manager.deleteSuccLid = manager.Chain().RegisterDeleteAccountBlocksSuccess(manager.onroadBlocksPool.RevertOnroadSuccess)
	manager.deleteOnRoadLid = manager.Chain().RegisterDeleteAccountBlocks(manager.onroadBlocksPool.RevertOnroad)
}

func (manager *Manager) Stop() {
	manager.log.Info("Close")
	manager.Net().UnsubscribeSyncStatus(manager.netStateLid)
	manager.keystoreManager.RemoveUnlockChangeChannel(manager.unlockLid)
	if manager.producer != nil {
		manager.Producer().SetAccountEventFunc(nil)
	}

	manager.Chain().UnRegister(manager.writeOnRoadLid)
	manager.Chain().UnRegister(manager.deleteOnRoadLid)
	manager.Chain().UnRegister(manager.writeSuccLid)
	manager.Chain().UnRegister(manager.deleteSuccLid)

	manager.stopAllWorks()
	manager.log.Info("Close end")
}

func (manager *Manager) Close() error {

	return nil
}

func (manager *Manager) netStateChangedFunc(state net.SyncState) {
	manager.log.Info("receive a net event", "state", state)
	if state == net.Syncdone {
		manager.resumeContractWorks()
	} else {
		manager.stopAllWorks()
	}
}

func (manager *Manager) addressLockStateChangeFunc(event keystore.UnlockEvent) {
	manager.log.Info("addressLockStateChangeFunc ", "event", event)

	w, found := manager.autoReceiveWorkers[event.Address]
	if found && !event.Unlocked() {
		manager.log.Info("found in autoReceiveWorkers stop it")
		common.Go(w.Stop)
	}
}

func (manager *Manager) producerStartEventFunc(accevent producerevent.AccountEvent) {
	netstate := manager.Net().SyncState()
	manager.log.Info("producerStartEventFunc receive event", "netstate", netstate)
	if netstate != net.Syncdone {
		return
	}

	event, ok := accevent.(producerevent.AccountStartEvent)
	if !ok {
		manager.log.Info("producerStartEventFunc not support this event")
		return
	}

	if !manager.keystoreManager.IsUnLocked(event.Address) {
		manager.log.Error("receive a right event but address locked", "event", event)
		return
	}

	manager.lastProducerAccEvent = &event

	w, found := manager.contractWorkers[event.Gid]
	if !found {
		w = NewContractWorker(manager)
		manager.contractWorkers[event.Gid] = w
	}

	nowTime := time.Now()
	if nowTime.After(event.Stime) && nowTime.Before(event.Etime) {
		w.Start(event)
		time.AfterFunc(event.Etime.Sub(nowTime), func() {
			w.Stop()
		})
	} else {
		w.Stop()
	}
}

func (manager *Manager) stopAllWorks() {
	manager.log.Info("stopAllWorks called")
	var wg = sync.WaitGroup{}
	for _, v := range manager.autoReceiveWorkers {
		wg.Add(1)
		common.Go(func() {
			v.Stop()
			wg.Done()
		})
	}
	for _, v := range manager.contractWorkers {
		wg.Add(1)
		common.Go(func() {
			v.Stop()
			wg.Done()
		})
	}
	wg.Wait()
	manager.log.Info("stopAllWorks end")
}

func (manager *Manager) resumeContractWorks() {
	manager.log.Info("resumeContractWorks")
	if manager.lastProducerAccEvent != nil {
		nowTime := time.Now()
		if nowTime.After(manager.lastProducerAccEvent.Stime) && nowTime.Before(manager.lastProducerAccEvent.Etime) {
			cw, ok := manager.contractWorkers[manager.lastProducerAccEvent.Gid]
			if ok {
				manager.log.Info("resumeContractWorks found an cw need to resume", "gid", manager.lastProducerAccEvent.Gid)
				cw.Start(*manager.lastProducerAccEvent)
				time.AfterFunc(manager.lastProducerAccEvent.Etime.Sub(nowTime), func() {
					cw.Stop()
				})
			}
		}
	}
	manager.log.Info("end resumeContractWorks")
}

func (manager *Manager) insertCommonBlockToPool(blockList []*vm_context.VmAccountBlock) error {
	return manager.pool.AddDirectAccountBlock(blockList[0].AccountBlock.AccountAddress, blockList[0])
}

func (manager *Manager) insertContractBlocksToPool(blockList []*vm_context.VmAccountBlock) error {
	if len(blockList) > 1 {
		return manager.pool.AddDirectAccountBlocks(blockList[0].AccountBlock.AccountAddress, blockList[0], blockList[1:])
	} else {
		return manager.pool.AddDirectAccountBlocks(blockList[0].AccountBlock.AccountAddress, blockList[0], nil)
	}
}

func (manager *Manager) checkExistInPool(addr types.Address, fromBlockHash types.Hash) bool {
	return manager.pool.ExistInPool(addr, fromBlockHash)
}

func (manager *Manager) ResetAutoReceiveFilter(addr types.Address, filter map[types.TokenTypeId]big.Int) {
	if w, ok := manager.autoReceiveWorkers[addr]; ok {
		w.ResetAutoReceiveFilter(filter)
	}
}

func (manager *Manager) StartAutoReceiveWorker(addr types.Address, filter map[types.TokenTypeId]big.Int) error {
	netstate := manager.Net().SyncState()
	manager.log.Info("StartAutoReceiveWorker ", "addr", addr, "netstate", netstate)

	if netstate != net.Syncdone {
		return ErrNotSyncDone
	}

	keystoreManager := manager.keystoreManager

	if _, e := keystoreManager.Find(addr); e != nil {
		return e
	}
	if !keystoreManager.IsUnLocked(addr) {
		return walleterrors.ErrLocked
	}

	w, found := manager.autoReceiveWorkers[addr]
	if !found {
		w = NewAutoReceiveWorker(manager, addr, filter)
		manager.log.Info("Manager get event new Worker")
		manager.autoReceiveWorkers[addr] = w
	}
	w.ResetAutoReceiveFilter(filter)
	w.Start()
	return nil
}

func (manager *Manager) StopAutoReceiveWorker(addr types.Address) error {
	manager.log.Info("StopAutoReceiveWorker ", "addr", addr)
	w, found := manager.autoReceiveWorkers[addr]
	if found {
		w.Stop()
	}
	return nil
}

func (manager Manager) ListWorkingAutoReceiveWorker() []types.Address {
	addr := make([]types.Address, 0)
	for _, v := range manager.autoReceiveWorkers {
		if v != nil && v.Status() == Start {
			addr = append(addr, v.address)
		}
	}

	return addr
}

func (manager Manager) GetOnroadBlocksPool() *model.OnroadBlocksPool {
	return manager.onroadBlocksPool
}

func (manager Manager) Chain() chain.Chain {
	return manager.chain
}

func (manager Manager) Net() Net {
	return manager.net
}

func (manager Manager) Producer() Producer {
	return manager.producer
}

func (manager Manager) DbAccess() *model.UAccess {
	return manager.uAccess
}
