package signer

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"sync"
	"time"
)

type Master struct {
	Vite                  Vite
	signSlaves            map[types.Address]*signSlave
	unlockEventListener   chan keystore.UnlockEvent
	FirstSyncDoneListener chan int
	coreMutex             sync.Mutex
	lid                   int
}

func NewMaster(vite Vite) *Master {
	return &Master{
		Vite:                  vite,
		signSlaves:            make(map[types.Address]*signSlave),
		unlockEventListener:   make(chan keystore.UnlockEvent),
		FirstSyncDoneListener: make(chan int),
		lid:                   0,
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
	syncinfo := c.Vite.Ledger().Sc().GetFirstSyncInfo()
	if !syncinfo.IsFirstSyncDone {
		log.Info("Master sync unfinished, so can't create transaction")
		return fmt.Errorf("master sync unfinished, so can't create transaction")
	}

	log.Info("Master AccountAddress", block.AccountAddress.String())
	log.Info("Master ToAddress", block.To.String())

	c.coreMutex.Lock()
	slave := c.signSlaves[*block.AccountAddress]
	endChannel := make(chan string, 1)

	if slave == nil {
		slave = NewsignSlave(c.Vite, *block.AccountAddress)
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

	c.Vite.Ledger().RegisterFirstSyncDown(c.FirstSyncDoneListener)
	go func() {
		log.Info("master waiting first sync done ")
		<-c.FirstSyncDoneListener
		close(c.FirstSyncDoneListener)
		log.Info("master first sync done ")
		c.lid = c.Vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(c.unlockEventListener)
		c.loop()
	}()
}

func (c *Master) loop() {
	status, _ := c.Vite.WalletManager().KeystoreManager.Status()
	for k, v := range status {
		if v == keystore.UnLocked {
			c.coreMutex.Lock()
			s := NewsignSlave(c.Vite, k)
			log.Info("Master find a new unlock address signSlave", k.String())
			c.signSlaves[k] = s
			c.coreMutex.Unlock()

			s.AddressUnlocked(true)
			go s.StartWork()
		}
	}
	for {
		event, ok := <-c.unlockEventListener
		log.Info("Master get event ", event)
		if !ok {
			log.Info("Master channel close ", event)
			c.Close()
		}

		c.coreMutex.Lock()
		if worker, ok := c.signSlaves[event.Address]; ok {
			log.Info("Master get event already exist ", event)
			c.coreMutex.Unlock()
			worker.AddressUnlocked(event.Unlocked())
			worker.newSignedTask <- struct{}{}
			continue
		}
		s := NewsignSlave(c.Vite, event.Address)
		log.Info("Master get event new signSlave")
		c.signSlaves[event.Address] = s
		c.coreMutex.Unlock()

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

	waitSendTasks   []*sendTask
	addressUnlocked bool
	isWorking       bool
	flagMutex       sync.Mutex
	isClosed        bool
}

func NewsignSlave(vite Vite, addrese types.Address) *signSlave {
	slave := &signSlave{vite: vite, address: addrese}
	slave.breaker = make(chan struct{}, 1)
	slave.newSignedTask = make(chan struct{}, 100)

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
		log.Info("slaver AddListener "+sw.address.String()+" sw.newSignedTask", sw.newSignedTask)
		sw.vite.Ledger().Ac().AddListener(sw.address, sw.newSignedTask)
	} else {
		log.Info("slaver RemoveListener", sw.address)
		sw.vite.Ledger().Ac().RemoveListener(sw.address)
	}
}

func (sw *signSlave) sendNextUnConfirmed() (hasmore bool, err error) {
	log.Info("slaver auto send confirm task", sw.address)
	ac := sw.vite.Ledger().Ac()
	hashes, e := ac.GetUnconfirmedTxHashs(0, 1, 1, &sw.address)

	if e != nil {
		log.Info("slaver auto GetUnconfirmedTxHashs err " + e.Error() + " " + sw.address.String())
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

	time.Sleep(2 * time.Second)
	return true, err
}

func (sw *signSlave) StartWork() {
	log.Info("slaver StartWork is called", sw.address.String())
	sw.flagMutex.Lock()
	if sw.isWorking {
		sw.flagMutex.Unlock()
		log.Info("slaver is working", sw.address.String())
		return
	}

	sw.isWorking = true

	sw.flagMutex.Unlock()
	log.Info("slaver start work", sw.address.String())
	for {
		log.Debug("slaver working")

		if sw.isClosed {
			break
		}
		if len(sw.waitSendTasks) != 0 {
			for i, v := range sw.waitSendTasks {
				log.Info("slaver send user task")
				err := sw.vite.Ledger().Ac().CreateTxWithPassphrase(v.block, v.passphrase)
				if err == nil {
					log.Info("slaver send user task success")
					v.end <- ""
				} else {
					log.Info("slaver send user task error", err.Error())
					v.end <- err.Error()
				}
				close(v.end)
				sw.waitSendTasks = append(sw.waitSendTasks[:i], sw.waitSendTasks[i+1:]...)
			}
		}

		if sw.addressUnlocked {
			hasmore, err := sw.sendNextUnConfirmed()
			if err != nil {
				log.Error(err.Error())
			}
			log.Info("slaver sendNextUnConfirmed hasmore", hasmore)
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
	sw.waitSendTasks = append(sw.waitSendTasks, task)

	log.Info(sw.address.String()+" is working ", sw.IsWorking())
	if sw.IsWorking() {
		go func() { sw.newSignedTask <- struct{}{} }()
	} else {
		go sw.StartWork()
	}
}
