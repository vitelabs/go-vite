package chain

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

func (c *Chain) GetNeedSnapshotContent() {

}

func (c *Chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {
	batch := new(leveldb.Batch)

	// Check and create account
	address := types.PubkeyToAddress(snapshotBlock.PublicKey)
	account, getErr := c.chainDb.Account.GetAccountByAddress(&address)

	if getErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+getErr.Error(), "method", "InsertSnapshotBlock")
		return getErr
	}

	if account == nil {
		// Create account
		c.createAccountLock.Lock()
		defer c.createAccountLock.Unlock()

		accountId, newAccountIdErr := c.newAccountId()
		if newAccountIdErr != nil {
			c.log.Error("newAccountId failed, error is "+newAccountIdErr.Error(), "method", "InsertSnapshotBlock")
			return newAccountIdErr
		}

		if err := c.createAccount(batch, accountId, &address, snapshotBlock.PublicKey); err != nil {
			c.log.Error("createAccount failed, error is "+getErr.Error(), "method", "InsertSnapshotBlock")
			return err
		}
	}

	// Save snapshot block
	if err := c.chainDb.Sc.WriteSnapshotBlock(batch, snapshotBlock); err != nil {
		c.log.Error("WriteSnapshotBlock failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Save snapshot content
	for _, accountBlockHashHeight := range snapshotBlock.SnapshotContent {
		accountBlockMeta, blockMetaErr := c.chainDb.Ac.GetBlockMeta(&accountBlockHashHeight.AccountBlockHash)
		if blockMetaErr != nil {
			c.log.Error("GetBlockMeta failed, error is "+blockMetaErr.Error(), "method", "InsertSnapshotBlock")
			return blockMetaErr
		}

		accountBlockMeta.SnapshotHeight = snapshotBlock.Height
		if saveSendBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlockHashHeight.AccountBlockHash, accountBlockMeta); saveSendBlockMetaErr != nil {
			c.log.Error("SaveBlockMeta failed, error is "+saveSendBlockMetaErr.Error(), "method", "InsertSnapshotBlock")
			return blockMetaErr
		}
	}

	if err := c.chainDb.Sc.WriteSnapshotContent(batch, snapshotBlock.Height, snapshotBlock.SnapshotContent); err != nil {
		c.log.Error("WriteSnapshotContent failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Save snapshot hash index
	c.chainDb.Sc.WriteSnapshotHash(batch, &snapshotBlock.Hash, snapshotBlock.Height)

	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Error("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Set cache
	c.latestSnapshotBlock = snapshotBlock
	// Delete needSnapshotCache
	for addr, item := range snapshotBlock.SnapshotContent {
		c.needSnapshotCache.Remove(&addr, item.AccountBlockHeight)
	}

	return nil
}
func (c *Chain) GetSnapshotBlocksByHash(originBlockHash *types.Hash, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	block, gsErr := c.GetSnapshotBlockByHash(originBlockHash)
	if gsErr != nil {
		c.log.Error("GetSnapshotBlockByHash failed, error is "+gsErr.Error(), "method", "GetSnapshotBlocksByHash")
		return nil, gsErr
	}
	if block == nil {
		return nil, nil
	}

	return c.GetSnapshotBlocksByHeight(block.Height, count, forward, containSnapshotContent)
}

func (c *Chain) GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	blocks, gErr := c.chainDb.Sc.GetSnapshotBlocks(height, count, forward, containSnapshotContent)
	if gErr != nil {
		c.log.Error("GetSnapshotBlocks failed, error is "+gErr.Error(), "method", "GetSnapshotBlocksByHeight")
		return nil, gErr
	}
	return blocks, gErr
}

func (c *Chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	block, gsbErr := c.chainDb.Sc.GetSnapshotBlock(height, true)
	if gsbErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+gsbErr.Error(), "method", "GetSnapshotBlockByHeight")
		return nil, gsbErr
	}

	if block != nil {
		snapshotContent, err := c.chainDb.Sc.GetSnapshotContent(block.Height)
		if err != nil {
			c.log.Error("GetSnapshotContent failed, error is "+err.Error(), "method", "GetSnapshotBlockByHeight")
			return nil, err
		}

		block.SnapshotContent = snapshotContent
	}

	return block, nil
}

func (c *Chain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	height, err := c.chainDb.Sc.GetSnapshotBlockHeight(hash)
	if err != nil {
		c.log.Error("GetSnapshotBlockHeight failed, error is "+err.Error(), "method", "GetSnapshotBlockByHash")
		return nil, err
	}
	if height <= 0 {
		return nil, nil
	}

	return c.GetSnapshotBlockByHeight(height)
}

func (c *Chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return c.latestSnapshotBlock
}

func (c *Chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return c.genesisSnapshotBlock
}

func (c *Chain) GetSbHashList(originBlockHash *types.Hash, count, step int, forward bool) ([]*types.Hash, error) {
	height, err := c.chainDb.Sc.GetSnapshotBlockHeight(originBlockHash)
	if err != nil {
		c.log.Error("GetSnapshotBlockHeight failed, error is "+err.Error(), "method", "GetSbHashList")
		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	if height <= 0 {
		return nil, nil
	}

	return c.chainDb.Sc.GetSbHashList(height, count, step, forward), nil
}

func (c *Chain) GetConfirmBlock(accountBlockHash *types.Hash) (*ledger.SnapshotBlock, error) {
	height, ghErr := c.chainDb.Ac.GetConfirmHeight(accountBlockHash)
	if ghErr != nil {
		c.log.Error("GetConfirmHeight failed, error is "+ghErr.Error(), "method", "GetConfirmBlock")
		return nil, &types.GetError{
			Code: 1,
			Err:  ghErr,
		}
	}

	snapshotBlock, gsErr := c.chainDb.Sc.GetSnapshotBlock(height, true)
	if gsErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+ghErr.Error(), "method", "GetConfirmBlock")
		return nil, &types.GetError{
			Code: 2,
			Err:  gsErr,
		}
	}

	return snapshotBlock, nil
}

func (c *Chain) GetConfirmTimes(accountBlockHash *types.Hash) (uint64, error) {
	height, ghErr := c.chainDb.Ac.GetConfirmHeight(accountBlockHash)
	if ghErr != nil {
		c.log.Error("GetConfirmHeight failed, error is "+ghErr.Error(), "method", "GetConfirmTimes")
		return 0, &types.GetError{
			Code: 1,
			Err:  ghErr,
		}
	}

	if height <= 0 {
		return 0, nil
	}

	return c.GetLatestSnapshotBlock().Height - height + 1, nil
}

func (c *Chain) GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error) {
	latestBlock := c.GetLatestSnapshotBlock()
	genesisBlock := c.GetGenesisSnapshotBlock()
	if latestBlock.Timestamp.Before(*blockCreatedTime) {
		return latestBlock, nil
	}

	if genesisBlock.Timestamp.After(*blockCreatedTime) {
		return nil, nil
	}

	blockCreatedUnixTime := blockCreatedTime.Unix()

	start := genesisBlock
	end := latestBlock

	for {
		if end.Height-start.Height <= 1 {
			var err error
			start.SnapshotContent, err = c.chainDb.Sc.GetSnapshotContent(start.Height)
			if err != nil {
				c.log.Error("GetSnapshotContent failed, error is "+err.Error(), "method", "GetSnapshotBlockBeforeTime")
				return nil, err
			}

			return start, nil
		}
		if end.Timestamp.Before(*start.Timestamp) {
			err := errors.New("end.Timestamp.Before(start.Timestamp)")
			return nil, err
		}

		nextEdgeHeight := start.Height + 1
		step := uint64(end.Timestamp.Unix()-start.Timestamp.Unix()) / (end.Height - start.Height)
		if step != 0 {
			startHeightGap := uint64(blockCreatedUnixTime-start.Timestamp.Unix()) / step
			if startHeightGap != 0 {
				nextEdgeHeight = start.Height + startHeightGap
			}
		}

		nextEdge, err := c.chainDb.Sc.GetSnapshotBlock(nextEdgeHeight, false)

		if err != nil {
			c.log.Error("Get try block failed, error is "+err.Error(), "method", "GetSnapshotBlockBeforeTime")
			return nil, err
		}

		if nextEdge.Timestamp.After(*blockCreatedTime) || nextEdge.Timestamp.Equal(*blockCreatedTime) {
			end = nextEdge
		} else {
			start = nextEdge
		}
	}
}

func (c *Chain) GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error) {
	account, getAccountIdErr := c.chainDb.Account.GetAccountByAddress(address)
	if getAccountIdErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+getAccountIdErr.Error(), "method", "GetConfirmAccountBlock")
		return nil, types.GetError{
			Code: 1,
			Err:  getAccountIdErr,
		}
	}
	if account == nil {
		return nil, nil
	}

	accountBlock, err := c.chainDb.Ac.GetConfirmAccountBlock(snapshotHeight, account.AccountId)
	if err != nil {
		c.log.Error("GetConfirmAccountBlock failed, error is "+err.Error(), "method", "GetConfirmAccountBlock")
		return nil, types.GetError{
			Code: 2,
			Err:  err,
		}
	}

	if accountBlock != nil {
		accountBlock.AccountAddress = account.AccountAddress
		// Not contract account block
		if len(accountBlock.PublicKey) == 0 {
			accountBlock.PublicKey = account.PublicKey
		}
		accountBlock.PublicKey = account.PublicKey
	}

	return accountBlock, nil
}

// TODO +toHeight judge
func (c *Chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	batch := new(leveldb.Batch)
	snapshotBlocks, accountBlocksMap, err := c.deleteSnapshotBlocksByHeight(batch, toHeight)
	if err != nil {
		c.log.Error("deleteSnapshotBlocksByHeight failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, nil, err
	}

	blockHeightMap := make(map[types.Address]uint64)
	for addr, accountBlocks := range accountBlocksMap {
		accountBlockHeight := accountBlocks[len(accountBlocks)-1].Height
		c.needSnapshotCache.Remove(&addr, accountBlockHeight)
		blockHeightMap[addr] = accountBlockHeight - 1
	}

	chainRangeSet := c.getChainRangeSet(snapshotBlocks)
	for addr, changeRangeItem := range chainRangeSet {
		blockHeightItem := blockHeightMap[addr]
		if blockHeightItem == 0 || blockHeightItem > changeRangeItem[0].Height-1 {
			blockHeightMap[addr] = changeRangeItem[0].Height - 1
		}
	}

	needAddBlocks := make(map[types.Address]*ledger.AccountBlock)
	for addr, blockHeight := range blockHeightMap {
		account, err := c.GetAccount(&addr)
		if err != nil {
			c.log.Error("GetAccount failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, nil, err
		}

		blockHash, blockHashErr := c.chainDb.Ac.GetHashByHeight(account.AccountId, blockHeight)
		if blockHashErr != nil {
			c.log.Error("GetHashByHeight failed, error is "+blockHashErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, nil, err
		}

		blockMeta, blockMetaErr := c.chainDb.Ac.GetBlockMeta(blockHash)
		if blockMetaErr != nil {
			c.log.Error("GetBlockMeta failed, error is "+blockMetaErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, nil, err
		}

		if blockMeta.SnapshotHeight <= 0 {
			block, blockErr := c.chainDb.Ac.GetBlockByHeight(account.AccountId, blockHeight)
			if blockErr != nil {
				c.log.Error("GetBlockByHeight failed, error is "+blockErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
				return nil, nil, err
			}
			block.AccountAddress = account.AccountAddress
			if len(block.PublicKey) == 0 {
				block.PublicKey = account.PublicKey
			}

			needAddBlocks[addr] = block
		}
	}

	if triggerErr := c.em.trigger(DeleteAccountBlocksEvent, accountBlocksMap); triggerErr != nil {
		c.log.Error("c.em.trigger, error is "+triggerErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, triggerErr
	}

	prevSnapshotBlock, prevSnapshotBlockErr := c.chainDb.Sc.GetSnapshotBlock(snapshotBlocks[0].Height-1, true)
	if prevSnapshotBlockErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+prevSnapshotBlockErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, prevSnapshotBlockErr
	}

	writeErr := c.chainDb.Commit(batch)
	if writeErr != nil {
		c.log.Error("Write db failed, error is "+writeErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, writeErr
	}

	// Set cache
	c.latestSnapshotBlock = prevSnapshotBlock

	// Set needSnapshotCache
	for addr, block := range needAddBlocks {
		c.needSnapshotCache.Add(&addr, block)
	}

	c.em.trigger(DeleteAccountBlocksSuccessEvent, accountBlocksMap)
	return snapshotBlocks, accountBlocksMap, nil
}

func (c *Chain) deleteSnapshotBlocksByHeight(batch *leveldb.Batch, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	maxAccountId, err := c.chainDb.Account.GetLastAccountId()
	if err != nil {
		c.log.Error("GetLastAccountId failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, err
	}

	planToDelete, getPlanErr := c.chainDb.Ac.GetPlanToDelete(maxAccountId, toHeight)
	if getPlanErr != nil {
		c.log.Error("GetPlanToDelete failed, error is "+getPlanErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
	}

	deleteMap, reopenList, getDeleteAndReopenErr := c.chainDb.Ac.GetDeleteMapAndReopenList(planToDelete, false)
	if getDeleteAndReopenErr != nil {
		c.log.Error("GetDeleteMapAndReopenList failed, error is "+getDeleteAndReopenErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, getDeleteAndReopenErr
	}

	deleteSnapshotBlocks, deleteSnapshotBlocksErr := c.chainDb.Sc.DeleteToHeight(batch, toHeight)
	if deleteSnapshotBlocksErr != nil {
		c.log.Error("DeleteByHeight failed, error is "+deleteSnapshotBlocksErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, deleteSnapshotBlocksErr
	}

	deleteAccountBlocks, deleteAccountBlocksErr := c.chainDb.Ac.Delete(batch, deleteMap)
	if deleteAccountBlocksErr != nil {
		c.log.Error("Delete failed, error is "+deleteAccountBlocksErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, deleteAccountBlocksErr
	}

	subLedger, toSubLedgerErr := c.subLedgerAccountIdToAccountAddress(deleteAccountBlocks)

	if toSubLedgerErr != nil {
		c.log.Error("subLedgerAccountIdToAccountAddress failed, error is "+toSubLedgerErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, toSubLedgerErr
	}

	reopenErr := c.chainDb.Ac.ReopenSendBlocks(batch, reopenList, deleteMap)
	if reopenErr != nil {
		c.log.Error("ReopenSendBlocks failed, error is "+reopenErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, reopenErr
	}

	return deleteSnapshotBlocks, subLedger, nil
}

func (c *Chain) getChainRangeSet(snapshotBlocks []*ledger.SnapshotBlock) map[types.Address][2]*ledger.HashHeight {
	chainRangeSet := make(map[types.Address][2]*ledger.HashHeight)
	for _, snapshotBlock := range snapshotBlocks {
		for addr, snapshotContent := range snapshotBlock.SnapshotContent {
			height := snapshotContent.AccountBlockHeight
			if chainRange := chainRangeSet[addr]; chainRange[0] == nil {
				chainRangeSet[addr] = [2]*ledger.HashHeight{
					{
						Hash:   snapshotContent.AccountBlockHash,
						Height: snapshotContent.AccountBlockHeight,
					}, {
						Hash:   snapshotContent.AccountBlockHash,
						Height: snapshotContent.AccountBlockHeight,
					},
				}
			} else if chainRange[0].Height > height {
				chainRange[0] = &ledger.HashHeight{
					Hash:   snapshotContent.AccountBlockHash,
					Height: snapshotContent.AccountBlockHeight,
				}
			} else if chainRange[1].Height < height {
				chainRange[1] = &ledger.HashHeight{
					Hash:   snapshotContent.AccountBlockHash,
					Height: snapshotContent.AccountBlockHeight,
				}
			}
		}
	}
	return chainRangeSet
}
