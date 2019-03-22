package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) DeleteSnapshotBlocks(toHash *types.Hash) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	height, err := c.indexDB.GetSnapshotBlockHeight(toHash)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockHeight failed, error is %s, snapshotHash is %s", err.Error(), toHash))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, nil, cErr
	}
	if height <= 0 {
		cErr := errors.New(fmt.Sprintf("height <= 0, error is %s, snapshotHash is %s", err.Error(), toHash))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, nil, cErr
	}

	return c.DeleteSnapshotBlocksToHeight(height)
}

func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	location, err := c.indexDB.GetSnapshotBlockLocation(toHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, snapshotHeight is %d", err.Error(), toHeight))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, nil, cErr
	}

	prevLocation, err := c.indexDB.GetSnapshotBlockLocation(toHeight - 1)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, (snapshotHeight -1) is %d", err.Error(), toHeight-1))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, nil, cErr
	}
	return c.deleteSnapshotBlocksToLocation(location, prevLocation)
}

func (c *chain) deleteSnapshotBlocksToLocation(location *chain_block.Location, prevLocation *chain_block.Location) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {

	// clean cache
	c.cache.CleanUnconfirmedPool()

	// delete blocks
	deletedSnapshotSegments, unconfirmedAccountBlocks, err := c.blockDB.DeleteTo(location, prevLocation)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.DeleteAndReadTo failed, error is %s, location is %d", err.Error(), location))
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
