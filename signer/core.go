package signer

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"sync"
)

type sendTask struct {
	block      *ledger.AccountBlock
	passphrase string
	end        chan string // err string
}

type signWorker struct {
	vite          *vite.Vite
	address       types.Address
	breaker       chan struct{}
	newSignedTask chan struct{}

	waitSendTasks   []*sendTask
	stopAutoConfirm bool
	isWorking       bool
	mutex           sync.Mutex
	isClosed        bool
}

func (sw *signWorker) Close() {

	sw.mutex.Lock()
	sw.isClosed = true
	sw.mutex.Unlock()

	sw.breaker <- struct{}{}
	close(sw.breaker)
}

func (sw *signWorker) disableAutoConfirm() {
	sw.stopAutoConfirm = true
}
func (sw *signWorker) sendNextUnConfirmed() bool {
	log.Info("auto send confirm task")
	return true
}

func (sw *signWorker) StartWork() {
	sw.mutex.Lock()
	if sw.isWorking {
		sw.mutex.Unlock()
		log.Info("worker %v is working", sw.address.String())
		return
	}

	sw.breaker = make(chan struct{}, 1)
	sw.newSignedTask = make(chan struct{}, 1)
	sw.isWorking = true

	log.Info("worker %v start work", sw.address.String())
	for {
		sw.mutex.Lock()
		if sw.isClosed {
			break
		}
		if len(sw.waitSendTasks) != 0 {
			for _, v := range sw.waitSendTasks {
				log.Info("send user task")
				err := sw.vite.Ledger().Ac.CreateTxWithPassphrase(v.block, v.passphrase)
				log.Info("send user task sucess")
				if err == nil {
					v.end <- ""
				} else {
					v.end <- err.Error()
				}
				close(v.end)
			}
		}
		sw.mutex.Unlock()

		if !sw.stopAutoConfirm {
			hasmore := sw.sendNextUnConfirmed()
			if hasmore {
				continue
			} else {
				goto WAIT
			}
		}

	WAIT:
		select {
		case <-sw.newSignedTask:
			continue
		case <-sw.breaker:
			break
		}

	}

	log.Info("worker %v end work", sw.address.String())
}

func (sw *signWorker) sendTask(task *sendTask) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	sw.waitSendTasks = append(sw.waitSendTasks, task)
}

type Core struct {
	vite                *vite.Vite
	signWorkers         map[types.Address]*signWorker
	unlockEventListener chan keystore.UnlockEvent
	mutex               sync.Mutex
}

// it is a sync func
func (c *Core) CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error {
	if block.AccountAddress == nil {
		return fmt.Errorf("address nil")
	}
	worker := c.signWorkers[*block.AccountAddress]
	endChannel := make(chan string, 1)

	if worker == nil {
		worker = &signWorker{vite: c.vite, address: *block.AccountAddress}
		c.signWorkers[*block.AccountAddress] = worker
	}

	worker.sendTask(&sendTask{
		block:      block,
		passphrase: passphrase,
		end:        endChannel,
	})
	err, ok := <-endChannel
	if !ok || err == "" {
		return nil
	}
	return fmt.Errorf(err)
}

func (c *Core) StartSign() {
	c.unlockEventListener = make(chan keystore.UnlockEvent)
	c.vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(c.unlockEventListener)
	go c.update()
}

func (c *Core) update() {
	for {
		event, ok := <-c.unlockEventListener
		if !ok {
			//todo close
		}

		c.mutex.Lock()
		defer c.mutex.Unlock()
		if worker, ok := c.signWorkers[event.Address]; ok {
			// if address locked the signwork need not care about confirm tx
			if !event.Unlocked() {
				worker.disableAutoConfirm()
			}
			continue
		}

		work := signWorker{vite: c.vite, address: event.Address}
		c.signWorkers[event.Address] = &work
		go work.StartWork()

	}
}
