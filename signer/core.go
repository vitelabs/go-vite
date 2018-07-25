package signer

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"sync"
)

type Master struct {
	Vite                Vite
	signSlaves          map[types.Address]*signSlave
	unlockEventListener chan keystore.UnlockEvent
	coreMutex           sync.Mutex
	lid                 int
}

func NewMaster(vite Vite) *Master {
	return &Master{
		Vite:                vite,
		signSlaves:          make(map[types.Address]*signSlave),
		unlockEventListener: make(chan keystore.UnlockEvent),
	}
}

func (c *Master) Close() error {
	log.Info("Master close")
	c.Vite.WalletManager().KeystoreManager.RemoveUnlockChangeChannel(c.lid)
	for _, v := range c.signSlaves {
		v.Close()
	}
	return nil

}
func (c *Master) CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error {
	if block.AccountAddress == nil {
		return fmt.Errorf("address nil")
	}

	c.coreMutex.Lock()
	slave := c.signSlaves[*block.AccountAddress]
	endChannel := make(chan string, 1)

	if slave == nil {
		slave = &signSlave{vite: c.Vite, address: *block.AccountAddress}
		c.signSlaves[*block.AccountAddress] = slave
	}
	c.coreMutex.Unlock()

	slave.sendTask(&sendTask{
		block:      block,
		passphrase: passphrase,
		end:        endChannel,
	})

	log.Info("sending Tx waiting ")
	err, ok := <-endChannel
	log.Info("sending Tx end err ", err)
	if !ok || err == "" {
		return nil
	}

	return fmt.Errorf(err)
}

func (c *Master) InitAndStartLoop() {
	c.lid = c.Vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(c.unlockEventListener)
	go c.loop()
}

func (c *Master) loop() {
	for {
		event, ok := <-c.unlockEventListener
		log.Info("Master get event ", event)
		if !ok {
			c.Close()
		}

		c.coreMutex.Lock()
		if worker, ok := c.signSlaves[event.Address]; ok {
			log.Info("Master get event already exist ", event)
			worker.AddressLocked(!event.Unlocked())
			continue
		}

		s := signSlave{vite: c.Vite, address: event.Address}
		log.Info("Master get event new signSlave")
		c.signSlaves[event.Address] = &s
		c.coreMutex.Unlock()

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
	return sw.isWorking
}

func (sw *signSlave) AddressLocked(locked bool) {
	sw.addressLocked = locked
	if locked {
		sw.vite.Ledger().Ac().AddListener(sw.address, sw.newSignedTask)
	} else {
		sw.vite.Ledger().Ac().RemoveListener(sw.address)
	}
}

func (sw *signSlave) sendNextUnConfirmed() (hasmore bool, err error) {
	log.Info("slaver auto send confirm task")
	ac := sw.vite.Ledger().Ac()
	hashes, e := ac.GetUnconfirmedTxHashs(0, 1, 1, &sw.address)

	if e != nil {
		return false, e
	}

	if len(hashes) == 0 {
		return false, nil
	}

	log.Info("slaver sendNextUnConfirmed: send receive transaction. " + sw.address.String() + " " + hashes[0].String())
	err = ac.CreateTx(&ledger.AccountBlock{
		AccountAddress: &sw.address,
		FromHash:       hashes[0],
	})

	return true, err
}

func (sw *signSlave) StartWork() {
	log.Info("slaver StartWork is called", sw.address.String())
	sw.mutex.Lock()
	if sw.isWorking {
		sw.mutex.Unlock()
		log.Info("slaver is working", sw.address.String())
		return
	}

	sw.breaker = make(chan struct{}, 1)
	sw.newSignedTask = make(chan struct{}, 100)

	sw.isWorking = true

	sw.mutex.Unlock()
	log.Info("slaver start work", sw.address.String())
	for {
		log.Debug("slaver working")
		sw.mutex.Lock()

		if sw.isClosed {
			break
		}
		if len(sw.waitSendTasks) != 0 {
			for _, v := range sw.waitSendTasks {
				log.Info("send user task")
				err := sw.vite.Ledger().Ac().CreateTxWithPassphrase(v.block, v.passphrase)
				if err == nil {
					log.Info("send user task success")
					v.end <- ""
				} else {
					log.Info("send user task error", err.Error())
					v.end <- err.Error()
				}
				close(v.end)
			}
		}
		sw.mutex.Unlock()

		if !sw.addressLocked {
			hasmore, err := sw.sendNextUnConfirmed()
			if err != nil {
				log.Error(err.Error())
			}
			if hasmore {
				continue
			} else {
				goto WAIT
			}
		}

	WAIT:
		log.Info("slaver start sleep", sw.address.String())
		select {
		case <-sw.newSignedTask:
			log.Info("slaver start awake", sw.address.String())
			continue
		case <-sw.breaker:
			log.Info("slaver start brake", sw.address.String())
			break
		}

	}

	sw.isWorking = false
	log.Info("slaver end work", sw.address.String())
}

func (sw *signSlave) sendTask(task *sendTask) {
	sw.mutex.Lock()
	sw.waitSendTasks = append(sw.waitSendTasks, task)
	sw.mutex.Unlock()

	log.Info(sw.address.String()+" is working ", sw.IsWorking())
	if sw.IsWorking() {
		go func() { sw.newSignedTask <- struct{}{} }()
	} else {
		go sw.StartWork()
	}
}
