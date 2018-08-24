package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"sync"
)

var slog = log15.New("module", "signer")

// todo 放到Miner模块
type RightEvent struct {
	gid       int
	address   types.Address
	event     string // start stop
	timestamp uint64
}

type Manager struct {
	Vite Vite

	workers map[types.Address]*Worker

	unlockEventListener   chan keystore.UnlockEvent
	firstSyncDoneListener chan int
	rightEventListener    chan RightEvent

	unlockLid    int
	rightLid     int
	firstSyncLid int

	mutex sync.Mutex
	log   log15.Logger
}

func NewManager(vite Vite) *Manager {
	return &Manager{
		Vite:                  vite,
		workers:               make(map[types.Address]*Worker),
		unlockEventListener:   make(chan keystore.UnlockEvent),
		firstSyncDoneListener: make(chan int),
		rightEventListener:    make(chan RightEvent),

		log: slog.New("w", "master"),
	}
}

func (manager *Manager) InitAndStartLoop() {
	manager.Vite.Ledger().RegisterFirstSyncDown(manager.firstSyncDoneListener)
	manager.unlockLid = manager.Vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(manager.unlockEventListener)
	// todo 注册Miner 监听器 manager.rightLid = manager.Vite.

	go manager.loop()

}

func (manager *Manager) Close() error {
	manager.log.Info("close")
	manager.Vite.WalletManager().KeystoreManager.RemoveUnlockChangeChannel(manager.unlockLid)
	// todo manager.Vite.Ledger().RemoveFirstSyncDownListener(manager.firstSyncDoneListener)
	for _, v := range manager.workers {
		v.Close()
	}
	return nil

}

func (manager *Manager) loop() {
	loopLog := manager.log.New("loop")

	status, _ := manager.Vite.WalletManager().KeystoreManager.Status()
	for k, v := range status {
		if v == keystore.UnLocked {
			manager.mutex.Lock()
			s := NewWorker(manager.Vite, k)
			loopLog.Info("Manager find a new unlock address ", "Worker", k.String())
			manager.workers[k] = s
			manager.mutex.Unlock()

			s.AddressUnlocked(true)
			go s.StartWork()
		}
	}

	for {
		select {
		case event, ok := <-manager.rightEventListener:
			{
				loopLog.Info("<-manager.rightEventListener ", "event", event)
				if !ok {
					manager.log.Info("Manager rightEvent channel close")
					break
				}

				worker, found := manager.workers[event.address]
				if !found || !worker.addressUnlocked {
					manager.log.Error(" receive a right event but address locked", "event", event)
					break
				}

				if event.event == "start" {
					worker.StartContractWork()
				} else {
					worker.StopContractWork()
				}

			}

		case done, ok := <-manager.firstSyncDoneListener:
			{
				loopLog.Info("<-manager.firstSyncDoneListener ", "done", done)
				if !ok {
					manager.log.Info("Manager firstSyncDoneListener channel close")
					break
				}

				for _, worker := range manager.workers {
					worker.IsFirstSyncDone(done)
				}

			}

		case event, ok := <-manager.unlockEventListener:
			{
				loopLog.Info("<-manager.unlockEventListener ", "event", event)
				if !ok {
					manager.log.Info("Manager channel close ")
					break
				}

				manager.mutex.Lock()
				if worker, ok := manager.workers[event.Address]; ok {
					manager.log.Info("get event already exist ", "event", event)
					manager.mutex.Unlock()
					worker.AddressUnlocked(event.Unlocked())
					worker.newSignedTask <- struct{}{}
					continue
				}
				s := NewWorker(manager.Vite, event.Address)
				loopLog.Info("Manager get event new Worker")
				manager.workers[event.Address] = s
				manager.mutex.Unlock()

				s.AddressUnlocked(event.Unlocked())
				go s.StartWork()
			}

		}

	}
}

