package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/worker"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"github.com/vitelabs/go-vite/wallet/walleterrors"
	"time"
)

var (
	slog = log15.New("module", "unconfirmed")
)

type Manager struct {
	Vite     worker.Vite
	dbAccess *UnconfirmedAccess

	commonTxWorkers map[types.Address]*worker.AutoReceiveWorker
	contractWorkers map[types.Address]*worker.ContractWorker

	unlockEventListener   chan keystore.UnlockEvent
	firstSyncDoneListener chan int
	rightEventListener    chan *RightEvent

	unlockLid    int
	rightLid     int
	firstSyncLid int

	log log15.Logger
}

type RightEvent struct {
	Gid            []byte
	Address        *types.Address
	StartTs        uint64
	EndTs          uint64
	SnapshotHash   *types.Hash
	SnapshotHeight int
}

func NewManager(vite worker.Vite) *Manager {
	return &Manager{
		Vite:            vite,
		commonTxWorkers: make(map[types.Address]*worker.AutoReceiveWorker),
		contractWorkers: make(map[types.Address]*worker.ContractWorker),

		unlockEventListener:   make(chan keystore.UnlockEvent),
		firstSyncDoneListener: make(chan int),
		rightEventListener:    make(chan *RightEvent),

		log: slog.New("w", "manager"),
	}
}

func (manager *Manager) InitAndStartWork() {
	manager.Vite.Ledger().RegisterFirstSyncDown(manager.firstSyncDoneListener)
	manager.unlockLid = manager.Vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(func(event keystore.UnlockEvent) {

	})
	// todo 注册Miner 监听器 manager.rightLid = manager.Vite.

	//todo add newContractListener????
	manager.dbAccess = NewUnconfirmedAccess(&manager.commonTxWorkers, &manager.contractWorkers)

	go func() {
		manager.initUnlockedAddress()
	}()
}

func (manager *Manager) Close() error {
	manager.log.Info("close")
	manager.Vite.WalletManager().KeystoreManager.RemoveUnlockChangeChannel(manager.unlockLid)
	// todo manager.Vite.Ledger().RemoveFirstSyncDownListener(manager.firstSyncDoneListener)
	for _, v := range manager.commonTxWorkers {
		v.Close()
	}
	for _, v := range manager.contractWorkers {
		v.Close()
	}
	return nil

}

func (manager *Manager) SetAutoReceiveFilter(f []worker.SimpleAutoReceiveFilterPair) {
	worker.SimpleAutoReceiveFilters = f
}

func (manager *Manager) StartAutoReceiveWorker(addr types.Address) error {
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
		w = worker.NewCommonTxWorker(manager.Vite, &addr)
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

// remove it in the future
func (manager *Manager) loop() {
	loopLog := manager.log.New("loop")

	for {
		select {
		case event, ok := <-manager.rightEventListener:
			{
				loopLog.Info("<-manager.rightEventListener ", "event", event)

				if !ok {
					manager.log.Info("Manager rightEvent channel close")
					break
				}

				if !manager.Vite.WalletManager().KeystoreManager.IsUnLocked(*event.Address) {
					manager.log.Error(" receive a right event but address locked", "event", event)
					break
				}

				w, found := manager.contractWorkers[*event.Address]
				if !found {
					addressList, err := manager.dbAccess.GetAddressListByGid(event.Gid)
					if err != nil || addressList == nil || len(addressList) < 0 {
						manager.log.Error("GetAddressListByGid Error", err)
						continue
					}
					w = worker.NewContractWorker(manager.Vite, manager.dbAccess, event.Gid, event.Address, addressList)
					manager.contractWorkers[*event.Address] = w

					break
				}
				nowTime := uint64(time.Now().UnixNano())
				if nowTime >= event.StartTs && nowTime < event.EndTs {
					w.Start(event)
				} else {
					w.Close()
				}
			}
		case done, ok := <-manager.firstSyncDoneListener:
			{
				loopLog.Info("<-manager.firstSyncDoneListener ", "done", done)
				if !ok {
					manager.log.Info("Manager firstSyncDoneListener channel close")
					break
				}

				//for _, worker := range manager.commonTxWorkers {
				//	worker.IsFirstSyncDone(done)
				//}

			}

			//case event, ok := <-manager.unlockEventListener:
			//	{
			//		loopLog.Info("<-manager.unlockEventListener ", "event", event)
			//		if !ok {
			//			manager.log.Info("Manager channel close ")
			//			break
			//		}
			//
			//		w, found := manager.commonTxWorkers[event.Address]
			//		if found {
			//			manager.log.Info("get event already exist ", "event", event)
			//			w.Start()
			//			//worker.AddressUnlocked(event.Unlocked())
			//			//worker.newSignedTask <- struct{}{}
			//			continue
			//		}
			//		w = worker.NewCommonTxWorker(manager.Vite, &event.Address)
			//		loopLog.Info("Manager get event new Worker")
			//		manager.commonTxWorkers[event.Address] = w
			//
			//		w.Start()
			//
			//	}

		}

	}
}

func (manager *Manager) initUnlockedAddress() {
	status, _ := manager.Vite.WalletManager().KeystoreManager.Status()
	for k, v := range status {
		if v == keystore.UnLocked {
			commonTxWorker := worker.NewCommonTxWorker(manager.Vite, &k)
			manager.log.Info("Manager find a new unlock address ", "Worker", k.String())
			manager.commonTxWorkers[k] = commonTxWorker
			commonTxWorker.Start()
		}
	}
}
