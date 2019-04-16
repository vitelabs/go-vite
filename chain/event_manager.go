package chain

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"sync"
)

type Listener interface{}

const (
	prepareInsertAbsEvent = byte(1)
	insertAbsEvent        = byte(2)

	prepareInsertSbsEvent = byte(3)
	InsertSbsEvent        = byte(4)

	prepareDeleteAbsEvent = byte(5)
	DeleteAbsEvent        = byte(6)

	prepareDeleteSbsEvent = byte(7)
	DeleteSbsEvent        = byte(8)
)

type eventManager struct {
	listenerList []EventListener

	maxHandlerId uint32
	mu           sync.Mutex
}

func newEventManager() *eventManager {
	return &eventManager{
		maxHandlerId: 0,
		listenerList: make([]EventListener, 0),
	}
}

func (em *eventManager) Trigger(eventType byte, vmAccountBlocks []*vm_db.VmAccountBlock,
	deleteAccountBlocks []*ledger.AccountBlock, snapshotBlocks []*ledger.SnapshotBlock, chunks []*ledger.SnapshotChunk) error {

	em.mu.Lock()
	defer em.mu.Unlock()

	if len(em.listenerList) <= 0 {
		return nil
	}

	switch eventType {
	case prepareInsertAbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareInsertAccountBlocks(vmAccountBlocks); err != nil {
				return err
			}
		}
	case insertAbsEvent:
		for _, listener := range em.listenerList {
			listener.InsertAccountBlocks(vmAccountBlocks)
		}

	case prepareInsertSbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareInsertSnapshotBlocks(snapshotBlocks); err != nil {
				return err
			}
		}
	case InsertSbsEvent:
		for _, listener := range em.listenerList {
			listener.InsertSnapshotBlocks(snapshotBlocks)
		}

	case prepareDeleteAbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareDeleteAccountBlocks(deleteAccountBlocks); err != nil {
				return err
			}
		}
	case DeleteAbsEvent:
		for _, listener := range em.listenerList {
			listener.DeleteAccountBlocks(deleteAccountBlocks)
		}

	case prepareDeleteSbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareDeleteSnapshotBlocks(chunks); err != nil {
				return err
			}
		}
	case DeleteSbsEvent:
		for _, listener := range em.listenerList {
			listener.DeleteSnapshotBlocks(chunks)
		}
	}

	return nil
}

func (em *eventManager) Register(listener EventListener) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.listenerList = append(em.listenerList, listener)
}
func (em *eventManager) UnRegister(listener EventListener) {
	em.mu.Lock()
	defer em.mu.Unlock()

	for index, listener := range em.listenerList {
		if listener == listener {
			em.listenerList = append(em.listenerList[:index], em.listenerList[index+1:]...)
			break
		}
	}
}

func (c *chain) Register(listener EventListener) {

	c.em.Register(listener)

}

func (c *chain) UnRegister(listener EventListener) {
	c.em.UnRegister(listener)
}
