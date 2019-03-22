package chain

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"sync"
	"sync/atomic"
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
	list map[byte]map[uint32]Listener

	prepareInsertAbsListenerList []PrepareInsertAccountBlocksListener
	insertAbsListenerList        []InsertAccountBlocksListener

	prepareInsertSbsListenerList []PrepareInsertSnapshotBlocksListener
	InsertSbsListenerList        []InsertSnapshotBlocksListener

	prepareDeleteAbsListenerList []PrepareDeleteAccountBlocksListener
	DeleteAbsListenerList        []DeleteAccountBlocksListener

	prepareDeleteSbsListenerList []PrepareDeleteSnapshotBlocksListener
	DeleteSbsListenerList        []PrepareDeleteSnapshotBlocksListener

	maxHandlerId uint32
	events       map[uint64]interface{}
	mu           sync.Mutex
}

func newEventManager() *eventManager {
	return &eventManager{
		maxHandlerId: 0,
		list:         make(map[byte]map[uint32]Listener, 8),
	}
}

func (em *eventManager) Trigger(eventType byte, vmAccountBlocks []*vm_db.VmAccountBlock,
	subLedger map[types.Address][]*ledger.AccountBlock, snapshotBlocks []*ledger.SnapshotBlock) {

	em.mu.Lock()
	defer em.mu.Unlock()

	listenerMap := em.list[eventType]

	if len(listenerMap) <= 0 {
		return
	}
	switch eventType {
	case prepareInsertAbsEvent:
		for _, listener := range listenerMap {
			listener.(PrepareInsertAccountBlocksListener)(vmAccountBlocks)
		}
	case insertAbsEvent:
		for _, listener := range listenerMap {
			listener.(InsertAccountBlocksListener)(vmAccountBlocks)
		}

	case prepareInsertSbsEvent:
		for _, listener := range listenerMap {
			listener.(PrepareInsertSnapshotBlocksListener)(snapshotBlocks)
		}
	case InsertSbsEvent:
		for _, listener := range listenerMap {
			listener.(InsertSnapshotBlocksListener)(snapshotBlocks)
		}

	case prepareDeleteAbsEvent:
		for _, listener := range listenerMap {
			listener.(PrepareDeleteAccountBlocksListener)(subLedger)
		}
	case DeleteAbsEvent:
		for _, listener := range listenerMap {
			listener.(DeleteAccountBlocksListener)(subLedger)
		}

	case prepareDeleteSbsEvent:
		for _, listener := range listenerMap {
			listener.(PrepareDeleteSnapshotBlocksListener)(snapshotBlocks, subLedger)
		}
	case DeleteSbsEvent:
		for _, listener := range listenerMap {
			listener.(DeleteSnapshotBlocksListener)(snapshotBlocks, subLedger)
		}
	}

}

func (em *eventManager) Register(eventType byte, listener Listener) uint64 {
	em.mu.Lock()
	defer em.mu.Unlock()

	eventHandlerId, handlerId := em.newEventHandlerId(eventType)

	if em.list[eventType] == nil {
		em.list[eventType] = make(map[uint32]Listener)
	}
	em.list[eventType][handlerId] = listener
	return eventHandlerId
}
func (em *eventManager) UnRegister(eventHandlerId uint64) {
	em.mu.Lock()
	defer em.mu.Unlock()

	eventHandlerIdBytes := make([]byte, 8)

	binary.BigEndian.PutUint64(eventHandlerIdBytes, eventHandlerId)

	eventType := eventHandlerIdBytes[3]
	if em.list[eventType] != nil {
		handlerId := binary.BigEndian.Uint32(eventHandlerIdBytes[4:])
		delete(em.list[eventType], handlerId)
	}
}

func (em *eventManager) newEventHandlerId(eventType byte) (uint64, uint32) {
	handlerId := atomic.AddUint32(&em.maxHandlerId, 1)
	eventHandlerId := make([]byte, 8)

	eventHandlerId[3] = eventType
	binary.BigEndian.PutUint32(eventHandlerId[4:], handlerId)

	return binary.BigEndian.Uint64(eventHandlerId), handlerId
}

func (c *chain) RegisterPrepareInsertAccountBlocks(listener PrepareInsertAccountBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(prepareInsertAbsEvent, listener)
}

func (c *chain) RegisterInsertAccountBlocks(listener InsertAccountBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(insertAbsEvent, listener)

}

func (c *chain) RegisterPrepareInsertSnapshotBlocks(listener PrepareInsertSnapshotBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(prepareInsertSbsEvent, listener)

}

func (c *chain) RegisterInsertSnapshotBlocks(listener InsertSnapshotBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(InsertSbsEvent, listener)

}

func (c *chain) RegisterPrepareDeleteAccountBlocks(listener PrepareDeleteAccountBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(prepareDeleteAbsEvent, listener)

}

func (c *chain) RegisterDeleteAccountBlocks(listener DeleteAccountBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(DeleteAbsEvent, listener)

}

func (c *chain) RegisterPrepareDeleteSnapshotBlocks(listener PrepareDeleteSnapshotBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(prepareDeleteSbsEvent, listener)

}

func (c *chain) RegisterDeleteSnapshotBlocks(listener DeleteSnapshotBlocksListener) (eventHandlerId uint64) {
	return c.em.Register(DeleteSbsEvent, listener)
}

func (c *chain) UnRegister(eventHandlerId uint64) {
	c.em.UnRegister(eventHandlerId)
}
