package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) DeleteSnapshotBlocks(toHash types.Hash) ([]*ledger.SnapshotChunk, error) {
	c.flusherMu.RLock()
	defer c.flusherMu.RUnlock()

	height, err := c.indexDB.GetSnapshotBlockHeight(&toHash)

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
	c.flusherMu.RLock()
	defer c.flusherMu.RUnlock()

	latestHeight := c.GetLatestSnapshotBlock().Height
	if toHeight > latestHeight {
		return nil, nil
	}

	snapshotChunkList := make([]*ledger.SnapshotChunk, 0, latestHeight-toHeight+1)

	var location *chain_file_manager.Location
	targetHeight := uint64(1)

	deletePerTime := uint64(1000)
	for targetHeight > toHeight {
		currentHeight := c.GetLatestSnapshotBlock().Height
		if currentHeight > deletePerTime {
			targetHeight = currentHeight - deletePerTime
		}
		if targetHeight < toHeight {
			targetHeight = toHeight
		}

		var err error
		location, err = c.indexDB.GetSnapshotBlockLocation(targetHeight)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, snapshotHeight is %d", err.Error(), toHeight))
			c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, cErr
		}

		chunkList, err := c.deleteSnapshotBlocksToLocation(location)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.deleteSnapshotBlocksToLocation failed, error is %s, toLocation is %+v", err.Error(), location))
			c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, cErr
		}
		snapshotChunkList = append(snapshotChunkList, chunkList...)
		//prevLocation, err := c.indexDB.GetSnapshotBlockLocation(toHeight - 1)
		//if err != nil {
		//	cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, error is %s, (snapshotHeight -1) is %d", err.Error(), toHeight-1))
		//	c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		//	return nil, cErr
		//}
	}

	return snapshotChunkList, nil
}

func (c *chain) deleteSnapshotBlocksToLocation(location *chain_file_manager.Location) ([]*ledger.SnapshotChunk, error) {

	// rollback blocks db
	deletedSnapshotSegments, err := c.blockDB.Rollback(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.DeleteAndReadTo failed, error is %s, location is %d", err.Error(), location))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback index db
	if err := c.indexDB.Rollback(deletedSnapshotSegments, location); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback cache
	err = c.cache.Rollback(deletedSnapshotSegments)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback state db
	if err := c.stateDB.Rollback(deletedSnapshotSegments); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	var snapshotChunkList = make([]*ledger.SnapshotChunk, 0, len(deletedSnapshotSegments))
	for _, seg := range deletedSnapshotSegments {
		snapshotChunkList = append(snapshotChunkList, &ledger.SnapshotChunk{
			SnapshotBlock: seg.SnapshotBlock,
			AccountBlocks: seg.AccountBlocks,
		})
	}

	//if len(deletedUnconfirmedBlocks) > 0 {
	//	snapshotChunkList = append(snapshotChunkList, &ledger.SnapshotChunk{
	//		AccountBlocks: deletedUnconfirmedBlocks,
	//	})
	//}
	return snapshotChunkList, nil
}
