package filters

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm_db"
)

type AccountChainEvent struct {
	BlockType     byte
	FromBlockHash types.Hash
	Hash          types.Hash
	Height        uint64
	Addr          types.Address
	ToAddr        types.Address
	Logs          []*ledger.VmLog
	SendBlockList []*SendBlock
}

type SendBlock struct {
	Hash   types.Hash
	ToAddr types.Address
}

func NewAccountChainEvent(block *ledger.AccountBlock, logs []*ledger.VmLog) *AccountChainEvent {
	ace := &AccountChainEvent{
		BlockType:     block.BlockType,
		FromBlockHash: block.FromBlockHash,
		Hash:          block.Hash,
		Height:        block.Height,
		Addr:          block.AccountAddress,
		ToAddr:        block.ToAddress,
		Logs:          logs}
	if length := len(block.SendBlockList); length > 0 {
		sendBlockList := make([]*SendBlock, length)
		for i, s := range block.SendBlockList {
			sendBlockList[i] = &SendBlock{Hash: s.Hash, ToAddr: s.ToAddress}
		}
		ace.SendBlockList = sendBlockList
	}
	return ace
}

type SnapshotChainEvent struct {
	Hash   types.Hash
	Height uint64
}

type ChainSubscribe struct {
	vite                   *vite.Vite
	es                     *EventSystem
	listenIdList           []uint64
	preDeleteAccountBlocks []*AccountChainEvent
}

func NewChainSubscribe(v *vite.Vite, e *EventSystem) *ChainSubscribe {
	c := &ChainSubscribe{vite: v, es: e}
	v.Chain().Register(c)
	return c
}

func (c *ChainSubscribe) Stop() {
	c.vite.Chain().UnRegister(c)
}

func (c *ChainSubscribe) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}

func (c *ChainSubscribe) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	acEvents := make([]*AccountChainEvent, len(blocks))
	for i, b := range blocks {
		acEvents[i] = NewAccountChainEvent(b.AccountBlock, b.VmDb.GetLogList())
	}
	c.es.acCh <- acEvents
	return nil
}

func (c *ChainSubscribe) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
func (c *ChainSubscribe) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	sbEvents := make([]*SnapshotChainEvent, 0, len(chunks))
	for _, chunk := range chunks {
		sbEvents = append(sbEvents, &SnapshotChainEvent{chunk.SnapshotBlock.Hash, chunk.SnapshotBlock.Height})
	}
	c.es.sbCh <- sbEvents
	return nil
}
func (c *ChainSubscribe) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	acEvents := make([]*AccountChainEvent, 0)
	for _, b := range blocks {
		if b.LogHash != nil {
			logList, err := c.vite.Chain().GetVmLogList(b.LogHash)
			if err != nil {
				c.es.log.Error("get log list failed when preDeleteAccountBlocks", "addr", b.AccountAddress, "hash", b.Hash, "height", b.Height, "err", err)
			}
			acEvents = append(acEvents, NewAccountChainEvent(b, logList))
		} else {
			acEvents = append(acEvents, NewAccountChainEvent(b, nil))
		}
	}
	c.preDeleteAccountBlocks = append(c.preDeleteAccountBlocks, acEvents...)
	return nil
}
func (c *ChainSubscribe) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	deletedBlocks := c.preDeleteAccountBlocks
	c.preDeleteAccountBlocks = nil
	c.es.acDelCh <- deletedBlocks
	return nil
}
func (c *ChainSubscribe) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	acEvents := make([]*AccountChainEvent, 0)
	for _, chunk := range chunks {
		for _, b := range chunk.AccountBlocks {
			if b.LogHash != nil {
				logList, err := c.vite.Chain().GetVmLogList(b.LogHash)
				if err != nil {
					c.es.log.Error("get log list failed when preDeleteSnapshotBlocks", "addr", b.AccountAddress, "hash", b.Hash, "height", b.Height, "err", err)
				}
				acEvents = append(acEvents, NewAccountChainEvent(b, logList))
			} else {
				acEvents = append(acEvents, NewAccountChainEvent(b, nil))
			}
		}
	}
	c.preDeleteAccountBlocks = append(c.preDeleteAccountBlocks, acEvents...)
	return nil
}
func (c *ChainSubscribe) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	sbEvents := make([]*SnapshotChainEvent, 0, len(chunks))
	for _, b := range chunks {
		if b.SnapshotBlock != nil {
			sbEvents = append(sbEvents, &SnapshotChainEvent{b.SnapshotBlock.Hash, b.SnapshotBlock.Height})
		}
	}
	c.es.sbDelCh <- sbEvents

	if len(c.preDeleteAccountBlocks) > 0 {
		deletedBlocks := c.preDeleteAccountBlocks
		c.preDeleteAccountBlocks = nil
		c.es.acDelCh <- deletedBlocks
	}
	return nil
}
