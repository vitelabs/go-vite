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
type AccountChainDelEvent struct {
}

type SnapshotChainEvent struct {
	SnapshotHash   types.Hash
	SnapshotHeight uint64
	Content        []*AccountChainEvent
}
type SnapshotChainDelEvent struct {
}

type ChainSubscribe struct {
	vite         *vite.Vite
	es           *EventSystem
	listenIdList []uint64
}

func NewChainSubscribe(v *vite.Vite, e *EventSystem) *ChainSubscribe {
	c := &ChainSubscribe{vite: v, es: e}
	list := make([]uint64, 0, 6)
	list = append(list, v.Chain().RegisterInsertAccountBlocksSuccess(c.InsertedAccountBlocks))
	// TODO list = append(list, v.Chain().RegisterInsertSnapshotBlocksSuccess(c.InsertedSnapshotBlocks))
	list = append(list, v.Chain().RegisterDeleteAccountBlocks(c.PreDeleteAccountBlocks))
	list = append(list, v.Chain().RegisterDeleteAccountBlocksSuccess(c.DeletedAccountBlocks))
	// TODO list = append(list, v.Chain().RegisterDeleteSnapshotBlocks(c.PreDeleteSnapshotBlocks))
	list = append(list, v.Chain().RegisterDeleteSnapshotBlocksSuccess(c.DeletedSnapshotBlocks))
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
	// TODO get blocks and logs detail and cache
	return nil
}
func (c *ChainSubscribe) DeletedAccountBlocks(subLedger map[types.Address][]*ledger.AccountBlock) {
	// TODO convert blocks and logs from cache and send to channel
}

func (c *ChainSubscribe) InsertedSnapshotBlocks(blocks []*ledger.SnapshotBlock, content map[types.Address][]*ledger.AccountBlock) {
	// TODO change content type
	var scEvents []*SnapshotChainEvent
	for _, b := range blocks {
		confirmedBlocks := content[b.Hash]
		if len(confirmedBlocks) == 0 {
			return
		}
		accList := make([]*AccountChainEvent, len(confirmedBlocks))
		for i, accB := range confirmedBlocks {
			if accB.LogHash != nil {
				logList, err := c.vite.Chain().GetVmLogList(accB.LogHash)
				if err != nil {
					c.es.log.Error("get vm log list failed when insert snapshot block", "hash", b.Hash, "err", err)
				}
				accList[i] = &AccountChainEvent{accB.Hash, accB.Height, accB.AccountAddress, logList}
			} else {
				accList[i] = &AccountChainEvent{accB.Hash, accB.Height, accB.AccountAddress, nil}
			}
		}
		scEvents = append(scEvents, &SnapshotChainEvent{b.Hash, b.Height, accList})
	}
	c.es.spCh <- scEvents
}

func (c *ChainSubscribe) PreDeleteSnapshotBlocks([]*ledger.SnapshotBlock) {
	// TODO
}
func (c *ChainSubscribe) DeletedSnapshotBlocks([]*ledger.SnapshotBlock) {
	// TODO
}

func (c *ChainSubscribe) GetLogs(addr types.Address, startSnapshotHeight uint64, endSnapshotHeight uint64) ([]*ledger.VmLog, error) {
	// TODO
	return nil, nil
}
