package pmchain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

/*
 *	1. prepare
 *	2.
 */
func (c *chain) InsertAccountBlock(vmAccountBlock *vm_context.VmAccountBlock) error {
	accountBlock := vmAccountBlock.AccountBlock
	// write unconfirmed pool
	if err := c.cache.UnconfirmedPool().InsertAccountBlock(accountBlock); err != nil {
		return err
	}

	// write index database, can query
	if err := c.indexDB.InsertAccountBlock(accountBlock); err != nil {
		return err
	}
	return nil
}

func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (subLedger map[types.Address][]*ledger.AccountBlock, err error) {
	return nil, nil
}
