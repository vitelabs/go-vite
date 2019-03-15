package pmchain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/vm_db"
)

/*
 *	1. prepare
 *	2.
 */
func (c *chain) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	accountBlock := vmAccountBlock.AccountBlock
	// write unconfirmed pool
	c.cache.InsertUnconfirmedAccountBlock(accountBlock)

	// write index database
	c.indexDB.InsertAccountBlock(vmAccountBlock)
	return nil
}

// no lock
func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (map[types.Address][]*ledger.AccountBlock, error) {
	unconfirmedBlocks := c.cache.GetCurrentUnconfirmedBlocks()
	canBeSnappedBlocks, invalidSubLedger := c.filterCanBeSnapped(unconfirmedBlocks)

	canBeSnappedSubLedger := blocksToMap(canBeSnappedBlocks)

	// remove unconfirmed subLedger
	c.cache.DeleteUnconfirmedSubLedger(canBeSnappedSubLedger)

	// write block db
	accountBlockLocations, snapshotBlockLocation, err := c.blockDB.Write(&chain_block.SnapshotSegment{
		SnapshotBlock: snapshotBlock,
		AccountBlocks: canBeSnappedBlocks,
	})
	if err != nil {
		chainErr := errors.New(fmt.Sprintf("c.blockDB.Write failed, error is %s, snapshotBlock is %+v", err.Error(), snapshotBlock))
		c.log.Error(chainErr.Error(), "method", "InsertSnapshotBlock")
		return nil, chainErr
	}

	// insert index
	if err := c.indexDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedSubLedger, snapshotBlockLocation, accountBlockLocations); err != nil {
		chainErr := errors.New(fmt.Sprintf("c.indexDB.InsertSnapshotBlock failed, error is %s, snapshotBlock is %+v", err.Error(), snapshotBlock))
		c.log.Error(chainErr.Error(), "method", "InsertSnapshotBlock")
		return nil, chainErr
	}

	// set cache
	c.cache.UpdateLatestSnapshotBlock(snapshotBlock)

	return invalidSubLedger, nil
}
