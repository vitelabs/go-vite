package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

/*
 *	1. prepare
 *	2.
 */
func (c *chain) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	vmAbList := []*vm_db.VmAccountBlock{vmAccountBlock}
	c.em.Trigger(prepareInsertAbsEvent, vmAbList, nil, nil, nil)

	accountBlock := vmAccountBlock.AccountBlock
	// write unconfirmed pool
	c.cache.InsertAccountBlock(accountBlock)

	// write index database
	if err := c.indexDB.InsertAccountBlock(accountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlock failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Crit(cErr.Error(), "method", "InsertAccountBlock")
	}

	// write state db
	if err := c.stateDB.Write(vmAccountBlock); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Write failed, error is %s, blockHash is %s", err.Error(), accountBlock.Hash))
		c.log.Crit(cErr.Error(), "method", "InsertAccountBlock")
	}

	c.em.Trigger(insertAbsEvent, vmAbList, nil, nil, nil)

	return nil
}

// no lock
func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) ([]*ledger.AccountBlock, error) {
	unconfirmedBlocks := c.cache.GetUnconfirmedBlocks()
	canBeSnappedBlocks, invalidAccountBlocks := c.filterCanBeSnapped(snapshotBlock.SnapshotContent, unconfirmedBlocks)

	sbList := []*ledger.SnapshotBlock{snapshotBlock}

	c.em.Trigger(prepareDeleteAbsEvent, nil, invalidAccountBlocks, nil, nil)
	c.em.Trigger(prepareInsertSbsEvent, nil, nil, sbList, nil)

	// write block db
	abLocationList, snapshotBlockLocation, err := c.blockDB.Write(&ledger.SnapshotChunk{
		SnapshotBlock: snapshotBlock,
		AccountBlocks: canBeSnappedBlocks,
	})

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Write failed, snapshotBlock is %+v. Error: %s", snapshotBlock, err.Error()))
		c.log.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	// insert index
	if err := c.indexDB.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks, snapshotBlockLocation, abLocationList, invalidAccountBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.InsertSnapshotBlock failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	// update latest snapshot block cache
	c.cache.InsertSnapshotBlock(snapshotBlock, canBeSnappedBlocks, invalidAccountBlocks)

	// flush state db
	if err := c.stateDB.InsertSnapshotBlock(invalidAccountBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.InsertSnapshotBlock failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "InsertSnapshotBlock")
	}

	c.flusher.Flush()

	c.em.Trigger(DeleteAbsEvent, nil, invalidAccountBlocks, nil, nil)
	c.em.Trigger(InsertSbsEvent, nil, nil, sbList, nil)

	return invalidAccountBlocks, nil
}
