package pmchain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

/*
 *	1. prepare
 *	2.
 */
func (c *chain) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	accountBlock := vmAccountBlock.AccountBlock
	// write unconfirmed pool
	if err := c.cache.UnconfirmedPool().InsertAccountBlock(accountBlock); err != nil {
		return err
	}

	// write index database
	c.indexDB.InsertAccountBlock(vmAccountBlock)
	return nil
}

func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (subLedger map[types.Address][]*ledger.AccountBlock, err error) {
	return nil, nil
}
