package chain

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"sync"
)

type insertProcessorFunc func(batch *leveldb.Batch, blocks []*vm_context.VmAccountBlock) error
type insertProcessorFuncSuccess func(blocks []*vm_context.VmAccountBlock)
type deleteProcessorFunc func(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error
type deleteProcessorFuncSuccess func(subLedger map[types.Address][]*ledger.AccountBlock)

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
			return listener.processor.(insertProcessorFunc)(data[0].(*leveldb.Batch), data[1].([]*vm_context.VmAccountBlock))
		case InsertAccountBlocksSuccessEvent:
			listener.processor.(insertProcessorFuncSuccess)(data[0].([]*vm_context.VmAccountBlock))
		case DeleteAccountBlocksEvent:
			return listener.processor.(deleteProcessorFunc)(data[0].(*leveldb.Batch), data[1].(map[types.Address][]*ledger.AccountBlock))
		case DeleteAccountBlocksSuccessEvent:
			listener.processor.(deleteProcessorFuncSuccess)(data[0].(map[types.Address][]*ledger.AccountBlock))
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

func (c *Chain) UnRegister(listenerId uint64) {
	c.em.unRegister(listenerId)
}

func (c *Chain) RegisterInsertAccountBlocks(processor insertProcessorFunc) uint64 {
	return c.em.register(InsertAccountBlocksEvent, processor)
}

func (c *Chain) RegisterInsertAccountBlocksSuccess(processor insertProcessorFuncSuccess) uint64 {
	return c.em.register(InsertAccountBlocksSuccessEvent, processor)
}

func (c *Chain) RegisterDeleteAccountBlocks(processor deleteProcessorFunc) uint64 {
	return c.em.register(DeleteAccountBlocksEvent, processor)
}

func (c *Chain) RegisterDeleteAccountBlocksSuccess(processor deleteProcessorFuncSuccess) uint64 {
	return c.em.register(DeleteAccountBlocksSuccessEvent, processor)
}
