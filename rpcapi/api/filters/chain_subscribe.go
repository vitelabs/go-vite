package filters

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm_context"
)

type AccountChainEvent struct {
	Hash   types.Hash
	Height uint64
	Addr   types.Address
	Logs   []*ledger.VmLog
}

type ChainSubscribe struct {
	vite                   *vite.Vite
	es                     *EventSystem
	listenIdList           []uint64
	preDeleteAccountBlocks []*AccountChainEvent
}

func NewChainSubscribe(v *vite.Vite, e *EventSystem) *ChainSubscribe {
	c := &ChainSubscribe{vite: v, es: e}
	list := make([]uint64, 0, 3)
	list = append(list, v.Chain().RegisterInsertAccountBlocksSuccess(c.InsertedAccountBlocks))
	list = append(list, v.Chain().RegisterDeleteAccountBlocks(c.PreDeleteAccountBlocks))
	list = append(list, v.Chain().RegisterDeleteAccountBlocksSuccess(c.DeletedAccountBlocks))
	c.listenIdList = list
	return c
}

func (c *ChainSubscribe) Stop() {
	for _, id := range c.listenIdList {
		c.vite.Chain().UnRegister(id)
	}
}

func (c *ChainSubscribe) InsertedAccountBlocks(blocks []*vm_context.VmAccountBlock) {
	acEvents := make([]*AccountChainEvent, len(blocks))
	for i, b := range blocks {
		acEvents[i] = &AccountChainEvent{b.AccountBlock.Hash, b.AccountBlock.Height, b.AccountBlock.AccountAddress, b.VmContext.GetLogList()}
	}
	c.es.acCh <- acEvents
}

func (c *ChainSubscribe) PreDeleteAccountBlocks(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error {
	acEvents := make([]*AccountChainEvent, 0)
	for addr, blocks := range subLedger {
		for _, b := range blocks {
			if b.LogHash != nil {
				logList, err := c.vite.Chain().GetVmLogList(b.LogHash)
				if err != nil {
					c.es.log.Error("get log list failed when preDeleteAccountBlocks", "addr", addr, "hash", b.Hash, "height", b.Height, "err", err)
				}
				acEvents = append(acEvents, &AccountChainEvent{b.Hash, b.Height, addr, logList})
			} else {
				acEvents = append(acEvents, &AccountChainEvent{b.Hash, b.Height, addr, nil})
			}
		}
	}
	c.preDeleteAccountBlocks = append(c.preDeleteAccountBlocks, acEvents...)
	return nil
}
func (c *ChainSubscribe) DeletedAccountBlocks(subLedger map[types.Address][]*ledger.AccountBlock) {
	deletedBlocks := c.preDeleteAccountBlocks
	c.preDeleteAccountBlocks = nil
	c.es.acDelCh <- deletedBlocks
}
