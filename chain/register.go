package chain

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"sync"
)

type insertProcessorFunc func(blocks []*vm_context.VmAccountBlock)
type deleteProcessorFunc func(blocks []*ledger.AccountBlock)

const (
	InsertAccountBlocks        = uint8(1)
	InsertAccountBlocksSuccess = uint8(2)

	DeleteAccountBlocks        = uint8(3)
	DeleteAccountBlocksSuccess = uint8(4)
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

func (em *eventManager) trigger(actionId uint8, data interface{}) {
	listenerList := em.eventListener[actionId]
	for _, listener := range listenerList {
		switch actionId {
		case InsertAccountBlocks:
			fallthrough
		case InsertAccountBlocksSuccess:
			listener.processor.(insertProcessorFunc)(data.([]*vm_context.VmAccountBlock))
		case DeleteAccountBlocks:
			fallthrough
		case DeleteAccountBlocksSuccess:
			listener.processor.(deleteProcessorFunc)(data.([]*ledger.AccountBlock))
		}
	}
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
	return c.em.register(InsertAccountBlocks, processor)
}

func (c *Chain) RegisterInsertAccountBlocksSuccess(processor insertProcessorFunc) uint64 {
	return c.em.register(InsertAccountBlocksSuccess, processor)
}

func (c *Chain) RegisterDeleteAccountBlocks(processor deleteProcessorFunc) uint64 {
	return c.em.register(DeleteAccountBlocks, processor)
}

func (c *Chain) RegisterDeleteAccountBlocksSuccess(processor deleteProcessorFunc) uint64 {
	return c.em.register(DeleteAccountBlocksSuccess, processor)
}
