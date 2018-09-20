package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"

	"errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"math/big"
	"sync"
	"time"
)

var (
	slog           = log15.New("module", "unconfirmed")
	ErrNotSyncDone = errors.New("network synchronization is not complete")
)

type Manager struct {
	vite            Vite
	keystoreManager *keystore.Manager

	pool                  PoolReader
	uAccess               *model.UAccess
	unconfirmedBlocksPool *model.UnconfirmedBlocksPool

	commonTxWorkers map[types.Address]*AutoReceiveWorker
	contractWorkers map[types.Gid]*ContractWorker

	unlockLid   int
	netStateLid int

	log log15.Logger
}

func NewManager(vite Vite, dataDir string) *Manager {
	m := &Manager{
		vite:            vite,
		pool:            vite,
		keystoreManager: vite.WalletManager().KeystoreManager,
		uAccess:         model.NewUAccess(vite.Chain(), dataDir),
		commonTxWorkers: make(map[types.Address]*AutoReceiveWorker),
		contractWorkers: make(map[types.Gid]*ContractWorker),
		log:             slog.New("w", "manager"),
	}
	m.unconfirmedBlocksPool = model.NewUnconfirmedBlocksPool(m.uAccess)
	return m
}

func (manager *Manager) InitAndStartWork() {
	manager.netStateLid = manager.vite.Net().SubscribeSyncStatus(manager.netStateChangedFunc)
	manager.unlockLid = manager.keystoreManager.AddLockEventListener(manager.addressLockStateChangeFunc)
	manager.vite.Producer().SetAccountEventFunc(manager.producerStartEventFunc)
}

func (manager *Manager) stopAllWorks() {
	var wg = sync.WaitGroup{}
	for _, v := range manager.commonTxWorkers {
		wg.Add(1)
		go func() {
			v.Stop()
			wg.Done()
		}()
	}
	for _, v := range manager.contractWorkers {
		wg.Add(1)
		go func() {
			v.Stop()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (manager *Manager) startAllWorks() {
	var wg = sync.WaitGroup{}
	for _, v := range manager.commonTxWorkers {
		wg.Add(1)
		go func() {
			v.Start()
			wg.Done()
		}()
	}
	for _, v := range manager.contractWorkers {
		wg.Add(1)
		go func() {
			v.Start()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (manager *Manager) Close() error {
	manager.log.Info("close")
	manager.vite.Net().UnsubscribeSyncStatus(manager.netStateLid)
	manager.keystoreManager.RemoveUnlockChangeChannel(manager.unlockLid)
	manager.vite.Producer().SetAccountEventFunc(nil)

	manager.stopAllWorks()
	manager.log.Info("close end")
	return nil
}

func (manager *Manager) netStateChangedFunc(state net.SyncState) {
	manager.log.Info("receive a net event", "state", state)
	if state == net.Syncdone {
		manager.startAllWorks()
	} else {
		manager.stopAllWorks()
	}
}

func (manager *Manager) addressLockStateChangeFunc(event keystore.UnlockEvent) {
	manager.log.Info("addressLockStateChangeFunc ", "event", event)

	w, found := manager.commonTxWorkers[event.Address]
	if found && !event.Unlocked() {
		manager.log.Info("found in commonTxWorkers stop it")
		go w.Stop()
	}
}

func (manager *Manager) producerStartEventFunc(accevent producer.AccountEvent) {
	netstate := manager.vite.Net().Status().SyncState
	manager.log.Info("producerStartEventFunc receive event", "netstate", netstate)
	if netstate != net.Syncdone {
		return
	}

	event, ok := accevent.(producer.AccountStartEvent)
	if !ok {
		manager.log.Info("producerStartEventFunc not support this event")
		return
	}

	if !manager.keystoreManager.IsUnLocked(event.Address) {
		manager.log.Error("receive a right event but address locked", "event", event)
		return
	}

	w, found := manager.contractWorkers[event.Gid]
	if !found {
		w, e := NewContractWorker(manager, event)
		if e != nil {
			manager.log.Error(e.Error())
			return
		}
		manager.contractWorkers[event.Gid] = w
	}

	nowTime := time.Now()
	if nowTime.After(event.Stime) && nowTime.Before(event.Etime) {
		w.Start()
	} else {
		w.Stop()
	}
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
	if w, ok := manager.commonTxWorkers[addr]; ok {
		w.ResetAutoReceiveFilter(filter)
	}
}

func (manager *Manager) StartAutoReceiveWorker(addr types.Address, filter map[types.TokenTypeId]big.Int) error {
	netstate := manager.vite.Net().Status().SyncState
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

	w, found := manager.commonTxWorkers[addr]
	if !found {
		w = NewAutoReceiveWorker(manager, addr, filter)
		manager.log.Info("Manager get event new Worker")
		manager.commonTxWorkers[addr] = w
	}
	w.Start()
	return nil
}

func (manager *Manager) StopAutoReceiveWorker(addr types.Address) error {
	manager.log.Info("StopAutoReceiveWorker ", "addr", addr)
	w, found := manager.commonTxWorkers[addr]
	if found {
		w.Stop()
	}
	return nil
}
