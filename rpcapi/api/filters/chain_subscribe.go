package filters

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm_context"
)

type AccountChainEvent struct {
	AccountBlock *ledger.AccountBlock
	Logs         []*ledger.VmLog
}
type AccountChainDelEvent struct {
}

type SnapshotChainEvent struct {
	SnapshotBlock *ledger.SnapshotBlock
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
	list = append(list, v.Chain().RegisterInsertSnapshotBlocksSuccess(c.InsertedSnapshotBlocks))
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
	// TODO convert blocks to event
	c.es.acCh <- AccountChainEvent{}
}

func (c *ChainSubscribe) PreDeleteAccountBlocks(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error {
	// TODO get blocks and logs detail and cache
	return nil
}
func (c *ChainSubscribe) DeletedAccountBlocks(subLedger map[types.Address][]*ledger.AccountBlock) {
	// TODO convert blocks and logs from cache and send to channel
}

func (c *ChainSubscribe) InsertedSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	// TODO
	c.es.spCh <- SnapshotChainEvent{blocks[0]}
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
