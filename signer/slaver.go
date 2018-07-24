package signer

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log"
	"sync"
)

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

	waitSendTasks []*sendTask
	addressLocked bool
	isWorking     bool
	mutex         sync.Mutex
	isClosed      bool
}

func (sw *signSlave) Close() error {

	sw.vite.Ledger().Ac().RemoveListener(sw.address)

	sw.mutex.Lock()
	sw.isClosed = true
	sw.isWorking = false
	sw.mutex.Unlock()

	sw.breaker <- struct{}{}
	close(sw.breaker)
	return nil
}

func (sw *signSlave) IsWorking() bool {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	return sw.isClosed
}

func (sw *signSlave) AddressLocked(locked bool) {
	sw.addressLocked = locked
	if locked {
		sw.vite.Ledger().Ac().AddListener(sw.address, sw.newSignedTask)
	} else {
		sw.vite.Ledger().Ac().RemoveListener(sw.address)
	}
}

func (sw *signSlave) sendNextUnConfirmed() bool {
	log.Info("auto send confirm task")
	return false
}

func (sw *signSlave) StartWork() {
	log.Info("slaver StartWork is called")
	sw.mutex.Lock()
	if sw.isWorking {
		sw.mutex.Unlock()
		log.Info("slaver %v is working", sw.address.String())
		return
	}

	sw.breaker = make(chan struct{}, 1)
	sw.newSignedTask = make(chan struct{}, 100)

	sw.isWorking = true

	sw.mutex.Unlock()
	log.Info("slaver %v start work", sw.address.String())
	for {
		sw.mutex.Lock()

		if sw.isClosed {
			break
		}
		if len(sw.waitSendTasks) != 0 {
			for _, v := range sw.waitSendTasks {
				log.Info("send user task")
				err := sw.vite.Ledger().Ac().CreateTxWithPassphrase(v.block, v.passphrase)
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

		if !sw.addressLocked {
			if sw.sendNextUnConfirmed() {
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

	sw.isWorking = false
	log.Info("slaver %v end work", sw.address.String())
}

func (sw *signSlave) sendTask(task *sendTask) {
	sw.mutex.Lock()
	sw.waitSendTasks = append(sw.waitSendTasks, task)
	sw.mutex.Unlock()

	if sw.IsWorking() {
		go func() { sw.newSignedTask <- struct{}{} }()
	} else {
		go sw.StartWork()
	}
}
