package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) DeleteSnapshotBlocks(toHash *types.Hash) ([]*ledger.SnapshotChunk, error) {
	height, err := c.indexDB.GetSnapshotBlockHeight(toHash)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockHeight failed, error is %s, snapshotHash is %s", err.Error(), toHash))
		c.log.Error(cErr.Error(), "method", "Rollback")
		return nil, cErr
	}
	if height <= 0 {
		cErr := errors.New(fmt.Sprintf("height <= 0, error is %s, snapshotHash is %s", err.Error(), toHash))
		c.log.Error(cErr.Error(), "method", "Rollback")
		return nil, cErr
	}

	return c.DeleteSnapshotBlocksToHeight(height)
}

func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error) {
	location, err := c.indexDB.GetSnapshotBlockLocation(toHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, snapshotHeight is %d", err.Error(), toHeight))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, cErr
	}

	prevLocation, err := c.indexDB.GetSnapshotBlockLocation(toHeight - 1)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, (snapshotHeight -1) is %d", err.Error(), toHeight-1))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, cErr
	}
	return c.deleteSnapshotBlocksToLocation(location, prevLocation)
}

func (c *chain) deleteSnapshotBlocksToLocation(
	location *chain_block.Location,
	prevLocation *chain_block.Location) ([]*ledger.SnapshotChunk, error) {

	// rollback blocks db
	deletedSnapshotSegments, unconfirmedAccountBlocks, newLatestSnapshotBlock, err := c.blockDB.Rollback(location, prevLocation)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.DeleteAndReadTo failed, error is %s, location is %d", err.Error(), location))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback index db
	if err := c.indexDB.Rollback(deletedSnapshotSegments, location); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback state db
	if err := c.stateDB.Rollback(deletedSnapshotSegments, location); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback cache
	deletedUnconfirmBlocks, err := c.cache.Rollback(deletedSnapshotSegments, unconfirmedAccountBlocks, newLatestSnapshotBlock)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	var snapshotChunkList = make([]*ledger.SnapshotChunk, 0, len(deletedSnapshotSegments)+1)
	for _, seg := range deletedSnapshotSegments {
		snapshotChunkList = append(snapshotChunkList, &ledger.SnapshotChunk{
			SnapshotBlock: seg.SnapshotBlock,
			AccountBlocks: seg.AccountBlocks,
		})
	}
	if len(deletedUnconfirmBlocks) > 0 {
		snapshotChunkList = append(snapshotChunkList, &ledger.SnapshotChunk{
			AccountBlocks: deletedUnconfirmBlocks,
		})
	}
	return snapshotChunkList, nil
}
