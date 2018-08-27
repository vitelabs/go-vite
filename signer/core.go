package signer

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"sync"
	"time"
)

var slog = log15.New("module", "signer")

type Master struct {
	Vite                  Vite
	signSlaves            map[types.Address]*signSlave
	unlockEventListener   chan keystore.UnlockEvent
	FirstSyncDoneListener chan int
	coreMutex             sync.Mutex
	lid                   int
	log                   log15.Logger
}

func NewMaster(vite Vite) *Master {
	return &Master{
		Vite:                  vite,
		signSlaves:            make(map[types.Address]*signSlave),
		unlockEventListener:   make(chan keystore.UnlockEvent),
		FirstSyncDoneListener: make(chan int),
		lid:                   0,
		log:                   slog.New("w", "master"),
	}
}

func (master *Master) Close() error {
	master.log.Info("close")
	master.Vite.WalletManager().KeystoreManager.RemoveUnlockChangeChannel(master.lid)
	for _, v := range master.signSlaves {
		v.Close()
	}
	return nil

}
func (master *Master) CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error {
	if block.AccountAddress == nil {
		return fmt.Errorf("address nil")
	}
	syncinfo := master.Vite.Ledger().Sc().GetFirstSyncInfo()
	if !syncinfo.IsFirstSyncDone {
		master.log.Info("sync unfinished, so can't create transaction")
		return fmt.Errorf("master sync unfinished, so can't create transaction")
	}

	master.log.Info("AccountAddress" + block.AccountAddress.String())
	master.log.Info("ToAddress" + block.To.String())

	master.coreMutex.Lock()
	slave := master.signSlaves[*block.AccountAddress]
	endChannel := make(chan string, 1)

	if slave == nil {
		slave = NewSignSlave(master.Vite, *block.AccountAddress)
		master.signSlaves[*block.AccountAddress] = slave
	}
	master.coreMutex.Unlock()

	slave.sendTask(&sendTask{
		block:      block,
		passphrase: passphrase,
		end:        endChannel,
	})

	master.log.Info("sending Tx waiting ")
	err, ok := <-endChannel
	master.log.Error("<-endChannel ", "err", err)
	if !ok || err == "" {
		return nil
	}

	return fmt.Errorf(err)
}

func (master *Master) InitAndStartLoop() {
	master.Vite.Ledger().RegisterFirstSyncDown(master.FirstSyncDoneListener)
	go func() {
		master.log.Info("master waiting first sync done ")
		<-master.FirstSyncDoneListener
		close(master.FirstSyncDoneListener)
		master.log.Info("<-master.firstSyncDoneListener first sync done ")
		master.lid = master.Vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(master.unlockEventListener)
		master.loop()
	}()
}

func (master *Master) loop() {
	loopLog := master.log.New("loop")
	status, _ := master.Vite.WalletManager().KeystoreManager.Status()
	for k, v := range status {
		if v == keystore.UnLocked {
			master.coreMutex.Lock()
			s := NewSignSlave(master.Vite, k)
			loopLog.Info("Master find a new unlock address ", "signSlave", k.String())
			master.signSlaves[k] = s
			master.coreMutex.Unlock()

			s.AddressUnlocked(true)
			go s.StartWork()
		}
	}
	for {
		event, ok := <-master.unlockEventListener
		loopLog.Info("<-master.unlockEventListener ", "event", event)
		if !ok {
			master.log.Info("Master channel close ", event)
			master.Close()
		}

		master.coreMutex.Lock()
		if worker, ok := master.signSlaves[event.Address]; ok {
			master.log.Info("get event already exist ", "event", event)
			master.coreMutex.Unlock()
			worker.AddressUnlocked(event.Unlocked())
			worker.newSignedTask <- struct{}{}
			continue
		}
		s := NewSignSlave(master.Vite, event.Address)
		loopLog.Info("Master get event new signSlave")
		master.signSlaves[event.Address] = s
		master.coreMutex.Unlock()

		s.AddressUnlocked(event.Unlocked())
		go s.StartWork()

	}
}

type sendTask struct {
	block      *ledger.AccountBlock
	passphrase string
	end        chan string // err string if string == "" means no error
}

type signSlave struct {
	vite          Vite
	address       types.Address
	breaker       chan struct{}
	newSignedTask chan struct{}
	log           log15.Logger

	waitSendTasks   []*sendTask
	addressUnlocked bool
	isWorking       bool
	flagMutex       sync.Mutex
	isClosed        bool
}

func NewSignSlave(vite Vite, addrese types.Address) *signSlave {
	slave := &signSlave{vite: vite,
		address: addrese,
		breaker: make(chan struct{}, 1),
		newSignedTask: make(chan struct{}, 100),
		log: slog.New("signSlave addr", addrese),
	}

	return slave
}

func (sw *signSlave) Close() error {

	sw.vite.Ledger().Ac().RemoveListener(sw.address)

	sw.flagMutex.Lock()
	sw.isClosed = true
	sw.isWorking = false
	sw.flagMutex.Unlock()

	sw.breaker <- struct{}{}
	close(sw.breaker)
	return nil
}

func (sw *signSlave) IsWorking() bool {
	sw.flagMutex.Lock()
	defer sw.flagMutex.Unlock()
	return sw.isWorking
}

func (sw *signSlave) AddressUnlocked(unlocked bool) {
	sw.addressUnlocked = unlocked
	if unlocked {
		sw.log.Info("AddressUnlocked", "sw.newSignedTask", sw.newSignedTask)
		sw.vite.Ledger().Ac().AddListener(sw.address, sw.newSignedTask)
	} else {
		sw.log.Info("AddressUnlocked RemoveListener ")
		sw.vite.Ledger().Ac().RemoveListener(sw.address)
	}
}

func (sw *signSlave) sendNextUnConfirmed() (hasmore bool, err error) {
	sw.log.Info("slaver auto send confirm task")
	ac := sw.vite.Ledger().Ac()
	hashes, e := ac.GetUnconfirmedTxHashs(0, 1, 1, &sw.address)

	if e != nil {
		sw.log.Error("slaver auto GetUnconfirmedTxHashs", "err", e)
		return false, e
	}

	if len(hashes) == 0 {
		return false, nil
	}

	sw.log.Info("slaver sendNextUnConfirmed: send receive transaction. ", "hash ", hashes[0])
	err = ac.CreateTx(&ledger.AccountBlock{
		AccountAddress: &sw.address,
		FromHash:       hashes[0],
	})

	time.Sleep(2 * time.Second)
	return true, err
}

func (sw *signSlave) StartWork() {

	sw.log.Info("slaver startWork is called")
	sw.flagMutex.Lock()
	if sw.isWorking {
		sw.flagMutex.Unlock()
		sw.log.Info("slaver is working")
		return
	}

	sw.isWorking = true

	sw.flagMutex.Unlock()
	sw.log.Info("slaver start work")
	for {
		sw.log.Debug("slaver working")

		if sw.isClosed {
			break
		}
		if len(sw.waitSendTasks) != 0 {
			for i, v := range sw.waitSendTasks {
				sw.log.Info("slaver send user task")
				err := sw.vite.Ledger().Ac().CreateTxWithPassphrase(v.block, v.passphrase)
				if err == nil {
					sw.log.Info("slaver send user task success")
					v.end <- ""
				} else {
					sw.log.Info("slaver send user task", "err", err)
					v.end <- err.Error()
				}
				close(v.end)
				sw.waitSendTasks = append(sw.waitSendTasks[:i], sw.waitSendTasks[i+1:]...)
			}
		}

		if sw.addressUnlocked {
			hasmore, err := sw.sendNextUnConfirmed()
			if err != nil {
				sw.log.Error(err.Error())
			}
			sw.log.Info("slaver sendNextUnConfirmed ", "hasmore", hasmore)
			if hasmore {
				continue
			} else {
				goto WAIT
			}
		}

	WAIT:
		sw.log.Info("slaver start sleep")
		select {
		case <-sw.newSignedTask:
			sw.log.Info("slaver start awake")
			continue
		case <-sw.breaker:
			sw.log.Info("slaver start brake")
			break
		}

	}

	sw.isWorking = false
	sw.log.Info("slaver end work")
}

func (sw *signSlave) sendTask(task *sendTask) {
	sw.waitSendTasks = append(sw.waitSendTasks, task)

	sw.log.Info("sendTask", "is working ", sw.IsWorking())
	if sw.IsWorking() {
		go func() { sw.newSignedTask <- struct{}{} }()
	} else {
		go sw.StartWork()
	}
}
