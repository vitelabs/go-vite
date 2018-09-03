package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/worker"
	"github.com/vitelabs/go-vite/wallet/keystore"
)

var slog = log15.New("module", "unconfirmed")

type Manager struct {
	Vite worker.Vite

	commonTxWorkers map[types.Address]*worker.CommonTxWorker
	contractWorkers map[types.Address]*worker.ContractWorker

	unlockEventListener   chan keystore.UnlockEvent
	firstSyncDoneListener chan int
	rightEventListener    chan RightEvent

	unlockLid    int
	rightLid     int
	firstSyncLid int

	log log15.Logger
}

type RightEvent struct {
	gid            string
	address        types.Address
	event          bool // "stop:true, start:false"
	timestamp      uint64
	snapshotHash   string
	snapshotHeight int
}

type RightEventArgs struct {
	gid            string
	address        types.Address
	timestamp      uint64
	snapshotHash   string
	snapshotHeight int
}

func NewRightEventArgs(gid string, address types.Address, timestamp uint64, snapshotHash string, snapshotHeight int) *RightEventArgs {
	return &RightEventArgs{
		gid:            gid,
		address:        address,
		timestamp:      timestamp,
		snapshotHash:   snapshotHash,
		snapshotHeight: snapshotHeight,
	}
}

func NewManager(vite worker.Vite) *Manager {
	return &Manager{
		Vite:            vite,
		commonTxWorkers: make(map[types.Address]*worker.CommonTxWorker),
		contractWorkers: make(map[types.Address]*worker.ContractWorker),

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
	for _, v := range manager.commonTxWorkers {
		v.Close()
	}
	for _, v := range manager.contractWorkers {
		v.Close()
	}
	return nil

}

func (manager *Manager) loop() {
	loopLog := manager.log.New("loop")

	status, _ := manager.Vite.WalletManager().KeystoreManager.Status()
	for k, v := range status {
		if v == keystore.UnLocked {
			commonTxWorker := worker.NewCommonTxWorker(manager.Vite, k)
			loopLog.Info("Manager find a new unlock address ", "Worker", k.String())
			manager.commonTxWorkers[k] = commonTxWorker
			commonTxWorker.Start()

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

				if !manager.Vite.WalletManager().KeystoreManager.IsUnLocked(event.address) {
					manager.log.Error(" receive a right event but address locked", "event", event)
					break
				}

				w, found := manager.contractWorkers[event.address]
				if !found {
					w = worker.NewContractWorker(manager.Vite, event.address, event.gid)
					manager.contractWorkers[event.address] = w
					break
				}

				select {
				case event.event == true:
					args := NewRightEventArgs(event.gid, event.address, event.timestamp, event.snapshotHash, event.snapshotHeight)
					w.Start(args)
				case event.event == false:
					w.Stop()
				default:
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

		case event, ok := <-manager.unlockEventListener:
			{
				loopLog.Info("<-manager.unlockEventListener ", "event", event)
				if !ok {
					manager.log.Info("Manager channel close ")
					break
				}

				w, found := manager.commonTxWorkers[event.Address]
				if found {
					manager.log.Info("get event already exist ", "event", event)
					w.Start()
					//worker.AddressUnlocked(event.Unlocked())
					//worker.newSignedTask <- struct{}{}
					continue
				}
				w = worker.NewCommonTxWorker(manager.Vite, event.Address)
				loopLog.Info("Manager get event new Worker")
				manager.commonTxWorkers[event.Address] = w

				w.Start()

			}

		}

	}
}
