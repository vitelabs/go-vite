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
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockHeight failed, snapshotHash is %s. Error: %s", toHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, cErr
	}
	if height <= 1 {
		cErr := errors.New(fmt.Sprintf("height <= 1,  snapshotHash is %s. Error: %s", toHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocks")
		return nil, cErr
	}

	return c.DeleteSnapshotBlocksToHeight(height)
}

// delete and recover unconfirmed cache
func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotChunk, error) {
	latestHeight := c.GetLatestSnapshotBlock().Height
	if toHeight > latestHeight || toHeight <= 1 {
		cErr := errors.New(fmt.Sprintf("toHeight is %d, latestHeight is %d", toHeight, latestHeight))
		c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, cErr
	}

	snapshotChunkList := make([]*ledger.SnapshotChunk, 0, latestHeight-toHeight+1)

	var location *chain_file_manager.Location

	targetHeight := latestHeight + 1

	deletePerTime := uint64(100)

	for targetHeight > toHeight {
		if targetHeight > deletePerTime {
			targetHeight = targetHeight - deletePerTime
			if targetHeight < toHeight {
				targetHeight = toHeight
			}
		} else {
			targetHeight = toHeight
		}

		var err error
		location, err = c.indexDB.GetSnapshotBlockLocation(targetHeight)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.indexDB.GetSnapshotBlockLocation failed, snapshotHeight is %d. Error: %s", targetHeight, err.Error()))
			c.log.Error(cErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, cErr
		}

		snapshotChunkList = append(snapshotChunkList, c.deleteSnapshotBlocksToLocation(location)...)
	}

	// rebuild unconfirmed cache
	if err := c.recoverUnconfirmedCache(); err != nil {
		return nil, err
	}

	c.flusher.Flush()

	return snapshotChunkList, nil
}

func (c *chain) deleteSnapshotBlocksToLocation(location *chain_file_manager.Location) []*ledger.SnapshotChunk {
	// rollback blocks db
	snapshotChunks, err := c.blockDB.Rollback(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Rollback failed, location is %d. Error: %s,", location, err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteSnapshotBlocksToLocation")
	}

	c.em.Trigger(prepareDeleteSbsEvent, nil, nil, nil, snapshotChunks)

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

	return snapshotChunks
}

func (c *chain) DeleteAccountBlocks(addr types.Address, toHash types.Hash) ([]*ledger.AccountBlock, error) {
	return c.deleteAccountBlocks(addr, 0, &toHash)
}

func (c *chain) DeleteAccountBlocksToHeight(addr types.Address, toHeight uint64) ([]*ledger.AccountBlock, error) {
	return c.deleteAccountBlocks(addr, toHeight, nil)
}

func (c *chain) deleteAccountBlocks(addr types.Address, toHeight uint64, toHash *types.Hash) ([]*ledger.AccountBlock, error) {
	unconfirmedBlocks := c.cache.GetUnconfirmedBlocks()
	if len(unconfirmedBlocks) <= 0 {
		cErr := errors.New(fmt.Sprintf("blocks is not unconfirmed, addr is %s, toHeight is %d", addr, toHeight))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlocks")
		return nil, cErr
	}
	var planDeleteBlocks []*ledger.AccountBlock
	for i, unconfirmedBlock := range unconfirmedBlocks {
		if (toHash != nil && unconfirmedBlock.Hash == *toHash) ||
			(toHeight > 0 && unconfirmedBlock.Height == toHeight) {
			planDeleteBlocks = unconfirmedBlocks[i:]
			break
		}
	}
	if len(planDeleteBlocks) <= 0 {
		cErr := errors.New(fmt.Sprintf("len(planDeleteBlocks) <= 0"))
		c.log.Error(cErr.Error(), "method", "deleteAccountBlocks")
		return nil, cErr
	}

	needDeleteBlocks := c.computeDependencies(planDeleteBlocks)

	if !c.stateDB.StorageRedo().HasRedo() {
		for _, block := range needDeleteBlocks {
			if ok, err := c.IsContractAccount(block.AccountAddress); err != nil {
				cErr := errors.New(fmt.Sprintf("c.IsContractAccount failed, addr is %s", block.AccountAddress))
				c.log.Error(cErr.Error(), "method", "deleteAccountBlocks")
				return nil, cErr
			} else if ok {
				// clean all, temporary implementation
				needDeleteBlocks = unconfirmedBlocks
				break
			}
		}
	}
	seg := []*ledger.SnapshotChunk{{
		AccountBlocks: needDeleteBlocks,
	}}

	c.em.Trigger(prepareDeleteAbsEvent, nil, needDeleteBlocks, nil, nil)

	// rollback index db
	if err := c.indexDB.Rollback(seg); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Rollback failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	// rollback cache
	if err := c.cache.RollbackAccountBlocks(needDeleteBlocks); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Rollback failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	// rollback state db
	if err := c.stateDB.Rollback(seg); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Rollback failed. Error: %s", err.Error()))
		c.log.Crit(cErr.Error(), "method", "deleteAccountBlocks")
	}

	c.em.Trigger(DeleteAbsEvent, nil, needDeleteBlocks, nil, nil)
	return needDeleteBlocks, nil
}
