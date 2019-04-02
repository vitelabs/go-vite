package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) DeleteSnapshotBlocks(toHash types.Hash) ([]*ledger.SnapshotChunk, error) {
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

// delete and recover unconfirmed cache
func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error) {
	latestHeight := c.GetLatestSnapshotBlock().Height
	if toHeight > latestHeight || toHeight <= 1 {
		return nil, nil
	}

	snapshotChunkList := make([]*ledger.SnapshotChunk, 0, latestHeight-toHeight+1)

	var location *chain_file_manager.Location
	targetHeight := uint64(1)

	deletePerTime := uint64(100)
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
	}

	// rebuild unconfirmed cache
	if err := c.recoverUnconfirmedCache(); err != nil {
		return nil, err
	}

	c.flusher.Flush()

	return snapshotChunkList, nil
}

func (c *chain) DeleteAccountBlocks(addr types.Address, toHash types.Hash) ([]*ledger.AccountBlock, error) {
	return c.deleteAccountBlocks(addr, 0, &toHash)
}

func (c *chain) DeleteAccountBlocksToHeight(addr types.Address, toHeight uint64) ([]*ledger.AccountBlock, error) {
	return c.deleteAccountBlocks(addr, toHeight, nil)
}

func (c *chain) deleteAccountBlocks(addr types.Address, toHeight uint64, toHash *types.Hash) ([]*ledger.AccountBlock, error) {
	unconfirmedBlocks := c.GetUnconfirmedBlocks(addr)
	if len(unconfirmedBlocks) <= 0 {
		cErr := errors.New(fmt.Sprintf("blocks is not unconfirmed, addr is %s, toHeight is %d", addr, toHeight))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlocks")
		return nil, cErr
	}
	var needDeleteBlocks []*ledger.AccountBlock
	for i, unconfirmedBlock := range unconfirmedBlocks {
		if (toHash != nil && unconfirmedBlock.Hash == *toHash) ||
			(toHeight > 0 && unconfirmedBlock.Height == toHeight) {
			needDeleteBlocks = unconfirmedBlocks[i:]
			break
		}
	}
	if len(needDeleteBlocks) <= 0 {
		cErr := errors.New(fmt.Sprintf("len(needDeleteBlocks) <= 0"))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlocks")
		return nil, cErr
	}

	blocks, err := c.findDependencies(needDeleteBlocks)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.findDependencies failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlocks")
		return nil, cErr
	}

	seg := []*ledger.SnapshotChunk{{
		AccountBlocks: blocks,
	}}

	c.em.Trigger(prepareDeleteAbsEvent, nil, blocks, nil, nil)

	// rollback index db
	if err := c.indexDB.Rollback(seg); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	// rollback cache
	err = c.cache.RollbackAccountBlocks(blocks)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	// rollback state db
	if err := c.stateDB.Rollback(seg); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	c.em.Trigger(DeleteAbsEvent, nil, blocks, nil, nil)
	return blocks, nil
}

func (c *chain) deleteSnapshotBlocksToLocation(location *chain_file_manager.Location) ([]*ledger.SnapshotChunk, error) {

	// rollback blocks db
	snapshotChunks, err := c.blockDB.Rollback(location)

	c.em.Trigger(prepareDeleteSbsEvent, nil, nil, nil, snapshotChunks)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.DeleteAndReadTo failed, error is %s, location is %d", err.Error(), location))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback index db
	if err := c.indexDB.Rollback(snapshotChunks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback cache
	err = c.cache.RollbackSnapshotBlocks(snapshotChunks)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	// rollback state db
	if err := c.stateDB.Rollback(snapshotChunks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Rollback failed, error is %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	c.flusher.Flush()

	c.em.Trigger(DeleteSbsEvent, nil, nil, nil, snapshotChunks)

	return snapshotChunks, nil
}
