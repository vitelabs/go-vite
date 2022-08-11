package chain

import (
	"sync"

	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
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
	deleteSbsEvent        = byte(8)
)

type eventManager struct {
	listenerList []interfaces.EventListener

	chain        *chain
	maxHandlerId uint32
	mu           sync.Mutex
}

func newEventManager(chain *chain) *eventManager {
	return &eventManager{
		chain:        chain,
		maxHandlerId: 0,
		listenerList: make([]interfaces.EventListener, 0),
	}
}

func (em *eventManager) TriggerInsertAbs(eventType byte, vmAccountBlocks []*interfaces.VmAccountBlock) error {
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
			err := listener.InsertAccountBlocks(vmAccountBlocks)
			if err != nil {
				em.chain.log.Error("Insert Account Block trigger fail", "err", err)
			}
		}
	}
	return nil
}

func (em *eventManager) TriggerDeleteAbs(eventType byte, accountBlocks []*ledger.AccountBlock) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if len(em.listenerList) <= 0 {
		return nil
	}

	switch eventType {
	case prepareDeleteAbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareDeleteAccountBlocks(accountBlocks); err != nil {
				return err
			}
		}
	case DeleteAbsEvent:
		for _, listener := range em.listenerList {
			err := listener.DeleteAccountBlocks(accountBlocks)
			if err != nil {
				em.chain.log.Error("Delete Account Block trigger fail", "err", err)
			}
		}
	}
	return nil
}

func splitChunks(chunks []*ledger.SnapshotChunk) ([]*ledger.SnapshotBlock, [][]*ledger.AccountBlock) {
	snapshotBlocks := make([]*ledger.SnapshotBlock, len(chunks))
	accountBlocksList := make([][]*ledger.AccountBlock, len(chunks))
	for i := 0; i < len(chunks); i++ {
		snapshotBlocks[i] = chunks[i].SnapshotBlock
		accountBlocksList[i] = chunks[i].AccountBlocks
	}

	return snapshotBlocks, accountBlocksList
}

func (em *eventManager) TriggerInsertSbs(eventType byte, chunks []*ledger.SnapshotChunk) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if len(em.listenerList) <= 0 {
		return nil
	}

	switch eventType {

	case prepareInsertSbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareInsertSnapshotBlocks(chunks); err != nil {
				return err
			}
		}

		splitChunks(chunks)
	case InsertSbsEvent:
		for _, listener := range em.listenerList {
			listener.InsertSnapshotBlocks(chunks)
		}

		splitChunks(chunks)
	}
	return nil
}

func (em *eventManager) TriggerDeleteSbs(eventType byte, chunks []*ledger.SnapshotChunk) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if len(em.listenerList) <= 0 {
		return nil
	}
	switch eventType {
	case prepareDeleteSbsEvent:
		for _, listener := range em.listenerList {
			if err := listener.PrepareDeleteSnapshotBlocks(chunks); err != nil {
				return err
			}
		}

		splitChunks(chunks)
	case deleteSbsEvent:
		for _, listener := range em.listenerList {
			listener.DeleteSnapshotBlocks(chunks)
		}

		splitChunks(chunks)
	}
	return nil
}

func (em *eventManager) Register(listener interfaces.EventListener) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.listenerList = append(em.listenerList, listener)
}

func (em *eventManager) UnRegister(listener interfaces.EventListener) {
	em.mu.Lock()
	defer em.mu.Unlock()

	for index, listener := range em.listenerList {
		if listener == listener {
			em.listenerList = append(em.listenerList[:index], em.listenerList[index+1:]...)
			break
		}
	}
}

func (c *chain) Register(listener interfaces.EventListener) {
	c.em.Register(listener)
}

func (c *chain) UnRegister(listener interfaces.EventListener) {
	c.em.UnRegister(listener)
}
