package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"sync"
)

const (
	InsertAccountBlocksEvent        = uint8(1)
	InsertAccountBlocksSuccessEvent = uint8(2)

	DeleteAccountBlocksEvent        = uint8(3)
	DeleteAccountBlocksSuccessEvent = uint8(4)
)

type listener struct {
	listenerId uint64
	processor  interface{}
}

type eventManager struct {
	eventListener map[uint8][]*listener

	maxListenerId uint64
	lock          sync.Mutex
}

func newEventManager() *eventManager {
	return &eventManager{
		eventListener: make(map[uint8][]*listener),

		maxListenerId: 0,
	}
}

func (em *eventManager) trigger(actionId uint8, data ...interface{}) error {
	listenerList := em.eventListener[actionId]
	for _, listener := range listenerList {
		switch actionId {
		case InsertAccountBlocksEvent:
			return listener.processor.(InsertProcessorFunc)(data[0].(*leveldb.Batch), data[1].([]*vm_context.VmAccountBlock))
		case InsertAccountBlocksSuccessEvent:
			listener.processor.(InsertProcessorFuncSuccess)(data[0].([]*vm_context.VmAccountBlock))
		case DeleteAccountBlocksEvent:
			return listener.processor.(DeleteProcessorFunc)(data[0].(*leveldb.Batch), data[1].(map[types.Address][]*ledger.AccountBlock))
		case DeleteAccountBlocksSuccessEvent:
			listener.processor.(DeleteProcessorFuncSuccess)(data[0].(map[types.Address][]*ledger.AccountBlock))
		}
	}
	return nil
}

func (em *eventManager) register(actionId uint8, processor interface{}) uint64 {
	em.lock.Lock()
	defer em.lock.Unlock()

	nextListenerId := em.maxListenerId + 1
	em.eventListener[actionId] = append(em.eventListener[actionId], &listener{
		listenerId: nextListenerId,
		processor:  processor,
	})
	return 0
}

func (em *eventManager) unRegister(listenerId uint64) {
	em.lock.Lock()
	defer em.lock.Unlock()

	for actionId, listenerList := range em.eventListener {
		for index, listener := range listenerList {
			if listener.listenerId == listenerId {
				em.eventListener[actionId] = append(listenerList[:index], listenerList[index+1:]...)
				return
			}
		}
	}
}

func (c *chain) UnRegister(listenerId uint64) {
	c.em.unRegister(listenerId)
}

func (c *chain) RegisterInsertAccountBlocks(processor InsertProcessorFunc) uint64 {
	return c.em.register(InsertAccountBlocksEvent, processor)
}

func (c *chain) RegisterInsertAccountBlocksSuccess(processor InsertProcessorFuncSuccess) uint64 {
	return c.em.register(InsertAccountBlocksSuccessEvent, processor)
}

func (c *chain) RegisterDeleteAccountBlocks(processor DeleteProcessorFunc) uint64 {
	return c.em.register(DeleteAccountBlocksEvent, processor)
}

func (c *chain) RegisterDeleteAccountBlocksSuccess(processor DeleteProcessorFuncSuccess) uint64 {
	return c.em.register(DeleteAccountBlocksSuccessEvent, processor)
}
