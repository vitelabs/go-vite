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
	if err := c.indexDB.InsertAccountBlock(vmAccountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlock failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Error(cErr.Error(), "method", "InsertAccountBlock")
		return cErr
	}
	return nil
}

// no lock
func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) (map[types.Address][]*ledger.AccountBlock, error) {
	unconfirmedBlocks := c.cache.GetCurrentUnconfirmedBlocks()
	canBeSnappedBlocks, invalidSubLedger, needDeletedAccountBlocks := c.filterCanBeSnapped(unconfirmedBlocks)

	canBeSnappedSubLedger := blocksToMap(canBeSnappedBlocks)

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

	// remove invalid subLedger index
	c.indexDB.DeleteInvalidAccountBlocks(invalidSubLedger)

	// remove invalid subLedger cache
	c.cache.DeleteUnconfirmedBlocks(needDeletedAccountBlocks)

	// update latest snapshot block cache
	c.cache.UpdateLatestSnapshotBlock(snapshotBlock)

	return invalidSubLedger, nil
}
