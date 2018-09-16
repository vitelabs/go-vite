package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/unconfirmed/worker"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"math/big"
	"time"
)

var (
	slog = log15.New("module", "unconfirmed")
)

type Manager struct {
	Vite                  Vite
	uAccess               *model.UAccess
	unconfirmedBlocksPool *model.UnconfirmedBlocksPool

	commonTxWorkers map[types.Address]*worker.AutoReceiveWorker
	contractWorkers map[types.Gid]*worker.ContractWorker

	unlockEventListener   chan keystore.UnlockEvent
	firstSyncDoneListener chan int
	rightEventListener    chan *producer.AccountStartEvent

	unlockLid    int
	rightLid     int
	firstSyncLid int

	log log15.Logger
}

func NewManager(vite Vite, dataDir string) *Manager {
	m := &Manager{
		Vite:            vite,
		commonTxWorkers: make(map[types.Address]*worker.AutoReceiveWorker),
		contractWorkers: make(map[types.Gid]*worker.ContractWorker),
		uAccess:         model.NewUAccess(vite.Chain(), dataDir),

		unlockEventListener:   make(chan keystore.UnlockEvent),
		firstSyncDoneListener: make(chan int),
		rightEventListener:    make(chan *producer.AccountStartEvent),

		log: slog.New("w", "manager"),
	}
	m.unconfirmedBlocksPool = model.NewUnconfirmedBlocksPool(m.uAccess)
	return m
}

func (manager *Manager) InitAndStartWork() {

	//manager.Vite.Chain().RegisterFirstSyncDown(manager.firstSyncDoneListener)

	manager.unlockLid = manager.Vite.WalletManager().KeystoreManager.AddLockEventListener(manager.addressLockStateChangeFunc)

	manager.Vite.Producer().SetAccountEventFunc(manager.producerStartEventFunc)

	//todo add newContractListener????

}

func (manager *Manager) Close() error {
	manager.log.Info("close")

	manager.Vite.WalletManager().KeystoreManager.RemoveUnlockChangeChannel(manager.unlockLid)

	manager.Vite.Producer().SetAccountEventFunc(nil)

	// todo manager.Vite.Ledger().RemoveFirstSyncDownListener(manager.firstSyncDoneListener)

	for _, v := range manager.commonTxWorkers {
		v.Close()
	}
	for _, v := range manager.contractWorkers {
		v.Close()
	}
	return nil

}

func (manager *Manager) SetAutoReceiveFilter(addr types.Address, filter map[types.TokenTypeId]big.Int) {
	if w, ok := manager.commonTxWorkers[addr]; ok {
		w.SetAutoReceiveFilter(filter)
	}
}

func (manager *Manager) StartAutoReceiveWorker(addr types.Address, filter map[types.TokenTypeId]big.Int) error {
	manager.log.Info("StartAutoReceiveWorker ", "addr", addr)

	keystoreManager := manager.Vite.WalletManager().KeystoreManager

	if _, e := keystoreManager.Find(addr); e != nil {
		return e
	}
	if !keystoreManager.IsUnLocked(addr) {
		return walleterrors.ErrLocked
	}

	w, found := manager.commonTxWorkers[addr]
	if !found {
		w = worker.NewAutoReceiveWorker(addr, filter)
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

func (manager *Manager) addressLockStateChangeFunc(event keystore.UnlockEvent) {
	manager.log.Info("addressLockStateChangeFunc ", "event", event)

	w, found := manager.commonTxWorkers[event.Address]
	if found && !event.Unlocked() {
		manager.log.Info("found in commonTxWorkers stop it")
		go w.Stop()
	}
}

func (manager *Manager) producerStartEventFunc(accevent producer.AccountEvent) {
	manager.log.Info("producerStartEventFunc receive event")
	event, ok := accevent.(producer.AccountStartEvent)
	if !ok {
		manager.log.Info("producerStartEventFunc not support this event")
		return
	}

	if !manager.Vite.WalletManager().KeystoreManager.IsUnLocked(event.Address) {
		manager.log.Error(" receive a right event but address locked", "event", event)
		return
	}

	w, found := manager.contractWorkers[event.Gid]
	if !found {
		w, e := worker.NewContractWorker(manager.unconfirmedBlocksPool, manager.Vite.WalletManager(), manager.Vite.Chain(), event)
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

// remove it in the future
func (manager *Manager) loop() {
	loopLog := manager.log.New("loop")

	for {
		select {
		case done, ok := <-manager.firstSyncDoneListener:
			{
				loopLog.Info("<-manager.firstSyncDoneListener ", "done", done)
				if !ok {
					manager.log.Info("Manager firstSyncDoneListener channel close")
					break
				}
			}
		}
	}
}

//func (manager *Manager) initUnlockedAddress() {
//	status, _ := manager.Vite.WalletManager().KeystoreManager.Status()
//	for k, v := range status {
//		if v == keystore.UnLocked {
//			commonTxWorker := worker.NewAutoReceiveWorker(manager.Vite, &k)
//			manager.log.Info("Manager find a new unlock address ", "Worker", k.String())
//			manager.commonTxWorkers[k] = commonTxWorker
//			commonTxWorker.Start()
//		}
//	}
//}
