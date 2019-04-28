package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) DeleteSnapshotBlocks(toHash types.Hash) ([]*ledger.SnapshotChunk, error) {
	height, err := c.indexDB.GetSnapshotBlockHeight(&toHash)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockHeight failed, snapshotHash is %s. Error: %s", toHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, cErr
	}
	if height <= 1 {
		cErr := errors.New(fmt.Sprintf("height <= 1, snapshotHash is %s. Error: %s", toHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, cErr
	}

	return c.DeleteSnapshotBlocksToHeight(height)
}

// delete and recover unconfirmed cache
func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error) {
	latestHeight := c.GetLatestSnapshotBlock().Height
	if toHeight > latestHeight || toHeight <= 1 {
		cErr := errors.New(fmt.Sprintf("toHeight is %d, GetLatestHeight is %d", toHeight, latestHeight))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, cErr
	}

	deleteAtOnce := uint64(120)
	// init target height
	targetHeight := latestHeight + 1

	allChunksDeleted := make([]*ledger.SnapshotChunk, 0, latestHeight-toHeight+1)

	for targetHeight > toHeight {
		// compute middle height to delete, because can't delete too much data at once
		if targetHeight > deleteAtOnce {
			targetHeight -= deleteAtOnce
			if targetHeight < toHeight {
				targetHeight = toHeight
			}
		} else {
			targetHeight = toHeight
		}

		// delete to middle height
		chunksDeleted, err := c.deleteSnapshotBlocksToHeight(targetHeight)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.deleteSnapshotBlocksToHeight failed, targetHeight is %d. Error: %s", targetHeight, err.Error()))
			c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, cErr
		}

		if len(allChunksDeleted) > 0 && allChunksDeleted[0].AccountBlocks == nil && chunksDeleted[len(chunksDeleted)-1].SnapshotBlock == nil {
			allChunksDeleted[0].AccountBlocks = chunksDeleted[len(chunksDeleted)-1].AccountBlocks
			allChunksDeleted = append(chunksDeleted[:len(chunksDeleted)-1], allChunksDeleted...)
		} else {
			allChunksDeleted = append(chunksDeleted, allChunksDeleted...)
		}

	}

	return allChunksDeleted, nil
}
func (c *chain) deleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error) {
	// lock flush
	c.flushMu.RLock()
	defer func() {
		c.flushMu.RUnlock()
		c.flusher.Flush()
	}()

	tmpLocation, err := c.indexDB.GetSnapshotBlockLocation(toHeight - 1)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, height is %d. Error: %s", toHeight-1, err.Error()))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
		return nil, cErr
	}

	location, err := c.blockDB.GetNextLocation(tmpLocation)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetNextLocation failed. Error: %s", err.Error()))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
		return nil, cErr
	}

	if location == nil {
		cErr := errors.New(fmt.Sprintf("location is nil, toHeight is %d",
			toHeight))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")

		return nil, cErr
	}

	// block db rollback
	snapshotChunks, err := c.blockDB.Rollback(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.RollbackAccountBlocks failed, location is %d. Error: %s,", location, err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
	}
	if len(snapshotChunks) <= 0 {
		return nil, nil
	}

	// rollback blocks db
	hasStorageRedoLog, err := c.stateDB.StorageRedo().HasRedo(toHeight)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.StorageRedo().HasRedo() failed, toHeight is %d. Error: %s", toHeight, err.Error()))
		c.log.Error(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
		return nil, cErr
	}

	var newUnconfirmedBlocks []*ledger.AccountBlock

	// append old unconfirmed blocks
	oldUnconfirmedBlocks := c.cache.GetUnconfirmedBlocks()
	if len(oldUnconfirmedBlocks) > 0 {
		snapshotChunks = append(snapshotChunks, &ledger.SnapshotChunk{
			AccountBlocks: oldUnconfirmedBlocks,
		})
	}

	realChunksToDelete := snapshotChunks

	if hasStorageRedoLog {
		newUnconfirmedBlocks = snapshotChunks[0].AccountBlocks

		// remove unconfirmed blocks
		realChunksToDelete = make([]*ledger.SnapshotChunk, len(snapshotChunks))
		copy(realChunksToDelete[1:], snapshotChunks[1:])

		firstChunk := *snapshotChunks[0]
		firstChunk.AccountBlocks = nil
		realChunksToDelete[0] = &firstChunk
	}

	//FOR DEBUG
	for _, chunk := range snapshotChunks {
		if chunk.SnapshotBlock != nil {
			c.log.Info(fmt.Sprintf("Delete snapshot block %d\n", chunk.SnapshotBlock.Height))
			for addr, sc := range chunk.SnapshotBlock.SnapshotContent {
				c.log.Info(fmt.Sprintf("Delete %d SC: %s %d %s\n", chunk.SnapshotBlock.Height, addr, sc.Height, sc.Hash))
			}
		}

		for _, ab := range chunk.AccountBlocks {
			c.log.Info(fmt.Sprintf("delete by sb %s %d %s\n", ab.AccountAddress, ab.Height, ab.Hash))
		}
	}

	//FOR DEBUG
	for _, block := range newUnconfirmedBlocks {
		c.log.Info(fmt.Sprintf("recover after delete sb %s %d %s\n", block.AccountAddress, block.Height, block.Hash))
	}

	if err := c.em.TriggerDeleteSbs(prepareDeleteSbsEvent, snapshotChunks); err != nil {
		return nil, err
	}

	// rollback index db
	if err := c.indexDB.RollbackSnapshotBlocks(snapshotChunks, newUnconfirmedBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.RollbackSnapshotBlocks failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
	}

	// rollback cache
	if err := c.cache.RollbackSnapshotBlocks(snapshotChunks, newUnconfirmedBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.RollbackSnapshotBlocks failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
	}

	// rollback state db
	if err := c.stateDB.RollbackSnapshotBlocks(snapshotChunks, newUnconfirmedBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.RollbackSnapshotBlocks failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
	}

	if err := c.em.TriggerDeleteSbs(DeleteSbsEvent, snapshotChunks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.em.Trigger(DeleteSbsEvent) failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToHeight")
	}

	return realChunksToDelete, nil
}

func (c *chain) DeleteAccountBlocks(addr types.Address, toHash types.Hash) ([]*ledger.AccountBlock, error) {
	return c.deleteAccountBlockByHeightOrHash(addr, 0, &toHash)
}

func (c *chain) DeleteAccountBlocksToHeight(addr types.Address, toHeight uint64) ([]*ledger.AccountBlock, error) {
	if toHeight <= 0 {
		return nil, errors.New("DeleteAccountBlocksToHeight failed, toHeight is 0")
	}
	return c.deleteAccountBlockByHeightOrHash(addr, toHeight, nil)
}

func (c *chain) deleteAccountBlockByHeightOrHash(addr types.Address, toHeight uint64, toHash *types.Hash) ([]*ledger.AccountBlock, error) {
	unconfirmedBlocks := c.cache.GetUnconfirmedBlocks()
	if len(unconfirmedBlocks) <= 0 {
		cErr := errors.New(fmt.Sprintf("blocks is not unconfirmed, Addr is %s, toHeight is %d", addr, toHeight))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlockByHeightOrHash")
		return nil, cErr
	}
	var planDeleteBlocks []*ledger.AccountBlock

	if toHash != nil {
		for i, unconfirmedBlock := range unconfirmedBlocks {
			if unconfirmedBlock.Hash == *toHash {
				planDeleteBlocks = unconfirmedBlocks[i:]

				break
			}
		}
	} else if toHeight > 0 {
		for i, unconfirmedBlock := range unconfirmedBlocks {
			if unconfirmedBlock.AccountAddress == addr && unconfirmedBlock.Height == toHeight {
				planDeleteBlocks = unconfirmedBlocks[i:]

				break
			}

		}
	}

	if len(planDeleteBlocks) <= 0 {
		cErr := errors.New(fmt.Sprintf("can't find block %s, %d, %s", addr, toHeight, toHash))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlockByHeightOrHash")
		return nil, cErr
	}

	needDeleteBlocks := c.computeDependencies(planDeleteBlocks)

	if err := c.deleteAccountBlocks(needDeleteBlocks); err != nil {
		return nil, err
	}

	return needDeleteBlocks, nil
}

func (c *chain) deleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// lock flush
	c.flushMu.RLock()
	defer c.flushMu.RUnlock()

	//FOR DEBUG
	for _, ab := range blocks {
		c.log.Info(fmt.Sprintf("delete by ab %s %d %s\n", ab.AccountAddress, ab.Height, ab.Hash))
	}

	if err := c.em.TriggerDeleteAbs(prepareDeleteAbsEvent, blocks); err != nil {
		return err
	}

	// rollback index db
	if err := c.indexDB.RollbackAccountBlocks(blocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.RollbackAccountBlocks failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	// rollback cache
	if err := c.cache.RollbackAccountBlocks(blocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.RollbackAccountBlocks failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	// rollback state db
	if err := c.stateDB.RollbackAccountBlocks(blocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.RollbackAccountBlocks failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	c.em.TriggerDeleteAbs(DeleteAbsEvent, blocks)
	return nil
}
