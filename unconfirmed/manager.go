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
	"time"
)

var (
	slog           = log15.New("module", "unconfirmed")
	ErrNotSyncDone = errors.New("net status has not sync done")
)

type Manager struct {
	vite                  Vite
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
		uAccess:         model.NewUAccess(vite.Chain(), dataDir),
		commonTxWorkers: make(map[types.Address]*AutoReceiveWorker),
		contractWorkers: make(map[types.Gid]*ContractWorker),
		log:             slog.New("w", "manager"),
	}
	m.unconfirmedBlocksPool = model.NewUnconfirmedBlocksPool(m.uAccess)
	return m
}

func (manager *Manager) InitAndStartWork() {
	manager.netStateLid = manager.vite.Net().SubscribeSyncStatus(manager.netStateChanged)
	manager.unlockLid = manager.vite.WalletManager().KeystoreManager.AddLockEventListener(manager.addressLockStateChangeFunc)
	manager.vite.Producer().SetAccountEventFunc(manager.producerStartEventFunc)
}

func (manager *Manager) stopAllWorks() {
	for _, v := range manager.commonTxWorkers {
		v.Close()
	}
	for _, v := range manager.contractWorkers {
		v.Close()
	}
}

func (manager *Manager) startAllWorks() {
	for _, v := range manager.commonTxWorkers {
		v.Start()
	}
	for _, v := range manager.contractWorkers {
		v.Start()
	}
}

func (manager *Manager) Close() error {
	manager.log.Info("close")

	manager.vite.Net().UnsubscribeSyncStatus(manager.netStateLid)
	manager.vite.WalletManager().KeystoreManager.RemoveUnlockChangeChannel(manager.unlockLid)
	manager.vite.Producer().SetAccountEventFunc(nil)

	manager.stopAllWorks()

	return nil
}

func (manager *Manager) netStateChanged(state net.SyncState) {
	manager.log.Info("receive a net evnet", "state", state)
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

	if !manager.vite.WalletManager().KeystoreManager.IsUnLocked(event.Address) {
		manager.log.Error(" receive a right event but address locked", "event", event)
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
		return manager.pool.AddDirectAccountBlocks(blockList[0].AccountBlock.AccountAddress,
			blockList[0], blockList[1:])
	} else {
		return manager.pool.AddDirectAccountBlocks(blockList[0].AccountBlock.AccountAddress,
			blockList[0], nil)
	}
}

func (manager *Manager) SetAutoReceiveFilter(addr types.Address, filter map[types.TokenTypeId]big.Int) {
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

	keystoreManager := manager.vite.WalletManager().KeystoreManager

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
