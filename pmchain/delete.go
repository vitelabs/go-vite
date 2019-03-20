package pmchain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (c *chain) DeleteSnapshotBlocks(toHash *types.Hash) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(toHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocationByHash failed, error is %s, snapshotHash is %s", err.Error(), toHash))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, nil, cErr
	}

	return c.deleteSnapshotBlocksToLocation(location)
}

func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	location, err := c.indexDB.GetSnapshotBlockLocation(toHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, snapshotHeight is %d", err.Error(), toHeight))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, nil, cErr
	}

	return c.deleteSnapshotBlocksToLocation(location)
}

func (c *chain) deleteSnapshotBlocksToLocation(location *chain_block.Location) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {

	// clean cache
	c.cache.CleanUnconfirmedPool()

	// delete blocks
	deletedSnapshotSegments, unconfirmedAccountBlocks, err := c.blockDB.DeleteTo(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.DeleteTo failed, error is %s, location is %d", err.Error(), location))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
		return nil, nil, cErr
	}

	// delete index
	if err := c.indexDB.DeleteSnapshotBlocks(deletedSnapshotSegments, unconfirmedAccountBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.DeleteSnapshotBlocks failed, error is %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
		return nil, nil, cErr
	}

	// delete state db
	if err := c.stateDB.DeleteSubLedger(deletedSnapshotSegments); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.DeleteSnapshotBlocks failed, error is %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
		return nil, nil, cErr
	}

	//insert unconfirmed block
	for _, block := range unconfirmedAccountBlocks {
		c.cache.InsertAccountBlock(block)
	}

	return nil, nil, nil
}
