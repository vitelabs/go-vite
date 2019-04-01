package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

/*
 *	1. prepare
 *	2.
 */
func (c *chain) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	c.flusherMu.RLock()
	defer c.flusherMu.RUnlock()

	vmAbList := []*vm_db.VmAccountBlock{vmAccountBlock}
	c.em.Trigger(prepareInsertAbsEvent, vmAbList, nil, nil)

	accountBlock := vmAccountBlock.AccountBlock
	// write unconfirmed pool
	c.cache.InsertAccountBlock(accountBlock)

	// write index database
	if err := c.indexDB.InsertAccountBlock(accountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlock failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Crit(cErr.Error(), "method", "InsertAccountBlock")
	}

	// write state_bak db
	if err := c.stateDB.Write(vmAccountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Write failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Crit(cErr.Error(), "method", "InsertAccountBlock")
	}

	c.em.Trigger(insertAbsEvent, vmAbList, nil, nil)
	return nil
}

// no lock
func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error) {
	c.flusherMu.RLock()
	defer c.flusherMu.RUnlock()

	sbList := []*ledger.SnapshotBlock{snapshotBlock}
	c.em.Trigger(prepareInsertSbsEvent, nil, nil, sbList)

	unconfirmedBlocks := c.cache.GetUnconfirmedBlocks()
	canBeSnappedBlocks, invalidAccountBlocks := c.filterCanBeSnapped(unconfirmedBlocks)

	// write block db
	abLocationList, snapshotBlockLocation, err := c.blockDB.Write(&chain_block.SnapshotSegment{
		SnapshotBlock: snapshotBlock,
		AccountBlocks: canBeSnappedBlocks,
	})

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Write failed, error is %s, snapshotBlock is %+v", err.Error(), snapshotBlock))
		c.log.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	// flush state db
	if err := c.stateDB.InsertSnapshotBlock(invalidAccountBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.InsertSnapshotBlock failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	// insert index
	c.indexDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks, snapshotBlockLocation, abLocationList, invalidAccountBlocks)

	// update latest snapshot block cache
	c.cache.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks, invalidAccountBlocks)

	c.em.Trigger(InsertSbsEvent, nil, nil, sbList)

	return invalidAccountBlocks, nil
}
