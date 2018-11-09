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

	InsertSnapshotBlocksSuccessEvent = uint8(6)

	DeleteSnapshotBlocksSuccessEvent = uint8(8)
)

type iabsListener struct {
	listenerId uint64
	processor  InsertProcessorFunc
}

type iabssListener struct {
	listenerId uint64
	processor  InsertProcessorFuncSuccess
}

type dabsListener struct {
	listenerId uint64
	processor  DeleteProcessorFunc
}
type dabssListener struct {
	listenerId uint64
	processor  DeleteProcessorFuncSuccess
}

type isbssListener struct {
	listenerId uint64
	processor  InsertSnapshotBlocksSuccess
}

type dsbssListener struct {
	listenerId uint64
	processor  DeleteSnapshotBlocksSuccess
}

type eventManager struct {
	iabsEventListener  []iabsListener
	iabssEventListener []iabssListener

	dabsEventListener  []dabsListener
	dabssEventListener []dabssListener

	isbssEventListener []isbssListener

	dsbssEventListener []dsbssListener

	maxListenerId uint64
	lock          sync.Mutex
}

func newEventManager() *eventManager {
	return &eventManager{
		maxListenerId: 0,
	}
}

func (em *eventManager) triggerInsertAccountBlocks(batch *leveldb.Batch, blocks []*vm_context.VmAccountBlock) error {
	for _, listener := range em.iabsEventListener {
		if err := listener.processor(batch, blocks); err != nil {
			return err
		}
	}
	return nil
}
func (em *eventManager) triggerInsertAccountBlocksSuccess(blocks []*vm_context.VmAccountBlock) {
	for _, listener := range em.iabssEventListener {
		listener.processor(blocks)
	}
}

func (em *eventManager) triggerDeleteAccountBlocks(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error {
	for _, listener := range em.dabsEventListener {
		if err := listener.processor(batch, subLedger); err != nil {
			return err
		}
	}
	return nil
}

func (em *eventManager) triggerDeleteAccountBlocksSuccess(subLedger map[types.Address][]*ledger.AccountBlock) {
	for _, listener := range em.dabssEventListener {
		listener.processor(subLedger)
	}
}

func (em *eventManager) triggerInsertSnapshotBlocksSuccess(snapshotBlocks []*ledger.SnapshotBlock) {
	for _, listener := range em.isbssEventListener {
		listener.processor(snapshotBlocks)
	}
}

func (em *eventManager) triggerDeleteSnapshotBlocksSuccess(snapshotBlocks []*ledger.SnapshotBlock) {
	for _, listener := range em.dsbssEventListener {
		listener.processor(snapshotBlocks)
	}
}

func (em *eventManager) register(actionId uint8, processor interface{}) uint64 {
	em.lock.Lock()
	defer em.lock.Unlock()

	nextListenerId := em.maxListenerId + 1
	switch actionId {
	case InsertAccountBlocksEvent:
		em.iabsEventListener = append(em.iabsEventListener, iabsListener{
			listenerId: nextListenerId,
			processor:  processor.(InsertProcessorFunc),
		})
	case InsertAccountBlocksSuccessEvent:
		em.iabssEventListener = append(em.iabssEventListener, iabssListener{
			listenerId: nextListenerId,
			processor:  processor.(InsertProcessorFuncSuccess),
		})
	case DeleteAccountBlocksEvent:
		em.dabsEventListener = append(em.dabsEventListener, dabsListener{
			listenerId: nextListenerId,
			processor:  processor.(DeleteProcessorFunc),
		})
	case DeleteAccountBlocksSuccessEvent:
		em.dabssEventListener = append(em.dabssEventListener, dabssListener{
			listenerId: nextListenerId,
			processor:  processor.(DeleteProcessorFuncSuccess),
		})
	case InsertSnapshotBlocksSuccessEvent:
		em.isbssEventListener = append(em.isbssEventListener, isbssListener{
			listenerId: nextListenerId,
			processor:  processor.(InsertSnapshotBlocksSuccess),
		})
	case DeleteSnapshotBlocksSuccessEvent:
		em.dsbssEventListener = append(em.dsbssEventListener, dsbssListener{
			listenerId: nextListenerId,
			processor:  processor.(DeleteSnapshotBlocksSuccess),
		})
	}

	return 0
}

func (em *eventManager) unRegister(listenerId uint64) {
	em.lock.Lock()
	defer em.lock.Unlock()

	for index, listener := range em.iabsEventListener {
		if listener.listenerId == listenerId {
			em.iabsEventListener = append(em.iabsEventListener[:index], em.iabsEventListener[index+1:]...)
			return
		}
	}

	for index, listener := range em.iabssEventListener {
		if listener.listenerId == listenerId {
			em.iabssEventListener = append(em.iabssEventListener[:index], em.iabssEventListener[index+1:]...)
			return
		}
	}

	for index, listener := range em.dabsEventListener {
		if listener.listenerId == listenerId {
			em.dabsEventListener = append(em.dabsEventListener[:index], em.dabsEventListener[index+1:]...)
			return
		}
	}
	for index, listener := range em.dabssEventListener {
		if listener.listenerId == listenerId {
			em.dabssEventListener = append(em.dabssEventListener[:index], em.dabssEventListener[index+1:]...)
			return
		}
	}

	for index, listener := range em.isbssEventListener {
		if listener.listenerId == listenerId {
			em.isbssEventListener = append(em.isbssEventListener[:index], em.isbssEventListener[index+1:]...)
			return
		}
	}

	for index, listener := range em.dsbssEventListener {
		if listener.listenerId == listenerId {
			em.dsbssEventListener = append(em.dsbssEventListener[:index], em.dsbssEventListener[index+1:]...)
			return
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

func (c *chain) RegisterInsertSnapshotBlocksSuccess(processor InsertSnapshotBlocksSuccess) uint64 {
	return c.em.register(InsertSnapshotBlocksSuccessEvent, processor)
}

func (c *chain) RegisterDeleteSnapshotBlocksSuccess(processor DeleteSnapshotBlocksSuccess) uint64 {
	return c.em.register(DeleteSnapshotBlocksSuccessEvent, processor)
}
