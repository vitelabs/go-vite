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

	err, ok := <-endChannel
	if !ok || err == "" {
		return nil
	}

	return fmt.Errorf(err)
}

func (c *Master) InitAndStartLoop() {
	c.unlockEventListener = make(chan keystore.UnlockEvent)
	c.lid = c.Vite.WalletManager().KeystoreManager.AddUnlockChangeChannel(c.unlockEventListener)
	go c.loop()
}

func (c *Master) loop() {
	for {
		event, ok := <-c.unlockEventListener
		log.Info("Master get event %v", event)
		if !ok {
			c.Close()
		}

		c.coreMutex.Lock()
		if worker, ok := c.signSlaves[event.Address]; ok {
			log.Info("Master get event already exist %v", event)
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
