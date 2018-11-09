package chain

import (
	"time"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
)

func (c *chain) GenStateTrie(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error) {
	prevTrie := c.GetStateTrie(&prevStateHash)
	if prevTrie == nil {
		prevTrie = c.NewStateTrie()
	}
	currentTrie := prevTrie.Copy()
	for addr, item := range snapshotContent {
		block := c.needSnapshotCache.GetBlockByHash(&addr, item.Hash)
		if block == nil {
			var err error
			block, err = c.chainDb.Ac.GetBlock(&item.Hash)
			if err != nil {
				c.log.Error("GetBlock failed, error is "+err.Error(), "method", "GenStateTrie")
				return nil, err
			}
		}

		if block != nil {
			currentTrie.SetValue(addr.Bytes(), block.StateHash.Bytes())
		}
	}

	return currentTrie, nil
}

func (c *chain) GetNeedSnapshotContent() ledger.SnapshotContent {
	return c.needSnapshotCache.GetSnapshotContent()
}

func (c *chain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {

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

		var caErr error
		if account, caErr = c.createAccount(batch, accountId, &address, snapshotBlock.PublicKey); caErr != nil {
			c.log.Error("createAccount failed, error is "+caErr.Error(), "method", "InsertSnapshotBlock")
			return caErr
		}
	}

	// Save snapshot block
	if err := c.chainDb.Sc.WriteSnapshotBlock(batch, snapshotBlock); err != nil {
		c.log.Error("WriteSnapshotBlock failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Save snapshot content
	for _, accountBlockHashHeight := range snapshotBlock.SnapshotContent {
		accountBlockMeta, blockMetaErr := c.chainDb.Ac.GetBlockMeta(&accountBlockHashHeight.Hash)
		if blockMetaErr != nil {
			c.log.Error("GetBlockMeta failed, error is "+blockMetaErr.Error(), "method", "InsertSnapshotBlock")
			return blockMetaErr
		}

		if saveSendBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlockHashHeight.Hash, accountBlockMeta); saveSendBlockMetaErr != nil {
			c.log.Error("SaveBlockMeta failed, error is "+saveSendBlockMetaErr.Error(), "method", "InsertSnapshotBlock")
			return blockMetaErr
		}

		if saveBeSnapshotErr := c.chainDb.Ac.WriteBeSnapshot(batch, &accountBlockHashHeight.Hash, snapshotBlock.Height); saveBeSnapshotErr != nil {
			c.log.Error("SaveBeSnapshot failed, error is "+saveBeSnapshotErr.Error(), "method", "InsertSnapshotBlock")
			return blockMetaErr
		}
	}

	if err := c.chainDb.Sc.WriteSnapshotContent(batch, snapshotBlock.Height, snapshotBlock.SnapshotContent); err != nil {
		c.log.Error("WriteSnapshotContent failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// Save snapshot hash index
	c.chainDb.Sc.WriteSnapshotHash(batch, &snapshotBlock.Hash, snapshotBlock.Height)

	// Save state trie
	var trieSaveCallback func()

	var saveTrieErr error
	if trieSaveCallback, saveTrieErr = snapshotBlock.StateTrie.Save(batch); saveTrieErr != nil {
		c.log.Error("Save state trie failed, error is "+saveTrieErr.Error(), "method", "InsertSnapshotBlock")
		return saveTrieErr
	}

	// Add snapshot block event
	c.chainDb.Be.AddSnapshotBlocks(batch, []types.Hash{snapshotBlock.Hash})

	// Delete needSnapshotCache, Need first update cache
	if c.needSnapshotCache != nil {
		c.needSnapshotCache.BeSnapshot(snapshotBlock.SnapshotContent)
	}

	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Crit("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertSnapshotBlock")
		return err
	}

	// After write db
	trieSaveCallback()

	// Set cache
	c.latestSnapshotBlock = snapshotBlock
	// Trigger success
	c.em.triggerInsertSnapshotBlocksSuccess([]*ledger.SnapshotBlock{snapshotBlock})

	// record insert
	c.blackBlock.InsertSnapshotBlocks([]*ledger.SnapshotBlock{snapshotBlock})

	// FIXME check
	for _, hashHeight := range snapshotBlock.SnapshotContent {
		accountBlockMeta, blockMetaErr := c.chainDb.Ac.GetBlockMeta(&hashHeight.Hash)
		if blockMetaErr != nil {
			c.log.Crit("GetBlockMeta failed, error is "+blockMetaErr.Error(), "method", "CheckInsertSnapshotBlock", "hash", hashHeight.Hash, "height", hashHeight.Height)
		}

		if accountBlockMeta == nil {
			c.log.Crit("AccountBlockMeta is nil.", "method", "CheckInsertSnapshotBlock", "hash", hashHeight.Hash, "height", hashHeight.Height)
		}

		if accountBlockMeta.SnapshotHeight <= 0 {
			c.log.Crit("AccountBlockMeta.SnapshotHeight <= 0.", "method", "CheckInsertSnapshotBlock", "hash", hashHeight.Hash, "height", hashHeight.Height)
		}
	}
	return nil
}

func (c *chain) GetSnapshotBlocksByHash(originBlockHash *types.Hash, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	startHeight := uint64(1)
	if originBlockHash != nil {
		block, gsErr := c.GetSnapshotBlockByHash(originBlockHash)
		if gsErr != nil {
			c.log.Error("GetSnapshotBlockByHash failed, error is "+gsErr.Error(), "method", "GetSnapshotBlocksByHash")
			return nil, gsErr
		}
		if block == nil {
			return nil, nil
		}
		startHeight = block.Height
	} else if !forward {
		block := c.GetLatestSnapshotBlock()
		startHeight = block.Height
	}

	return c.GetSnapshotBlocksByHeight(startHeight, count, forward, containSnapshotContent)
}

func (c *chain) GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error) {
	blocks, gErr := c.chainDb.Sc.GetSnapshotBlocks(height, count, forward, containSnapshotContent)
	if gErr != nil {
		c.log.Error("GetSnapshotBlocks failed, error is "+gErr.Error(), "method", "GetSnapshotBlocksByHeight")
		return nil, gErr
	}
	return blocks, gErr
}

func (c *chain) GetSnapshotBlockHeadByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	block, gsbErr := c.chainDb.Sc.GetSnapshotBlock(height, false)
	if gsbErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+gsbErr.Error(), "method", "GetSnapshotBlockHeadByHeight")
		return nil, gsbErr
	}

	return block, nil
}

func (c *chain) GetSnapshotBlockHeadByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	height, err := c.chainDb.Sc.GetSnapshotBlockHeight(hash)
	if err != nil {
		c.log.Error("GetSnapshotBlockHeight failed, error is "+err.Error(), "method", "GetSnapshotBlockHeadByHash")
		return nil, err
	}
	if height <= 0 {
		return nil, nil
	}

	return c.GetSnapshotBlockHeadByHeight(height)
}

func (c *chain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	block, gsbErr := c.chainDb.Sc.GetSnapshotBlock(height, true)
	if gsbErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+gsbErr.Error(), "method", "GetSnapshotBlockByHeight")
		return nil, gsbErr
	}

	return block, nil
}

func (c *chain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
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

func (c *chain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	return c.latestSnapshotBlock
}

func (c *chain) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return c.genesisSnapshotBlock
}

func (c *chain) GetConfirmBlock(accountBlockHash *types.Hash) (*ledger.SnapshotBlock, error) {
	height, ghErr := c.chainDb.Ac.GetConfirmHeight(accountBlockHash)
	if ghErr != nil {
		c.log.Error("GetConfirmHeight failed, error is "+ghErr.Error(), "method", "GetConfirmBlock")
		return nil, ghErr
	}

	if height <= 0 {
		return nil, nil
	}

	snapshotBlock, gsErr := c.chainDb.Sc.GetSnapshotBlock(height, true)
	if gsErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+gsErr.Error(), "method", "GetConfirmBlock")
		return nil, gsErr
	}

	return snapshotBlock, nil
}

func (c *chain) GetConfirmTimes(accountBlockHash *types.Hash) (uint64, error) {
	height, ghErr := c.chainDb.Ac.GetConfirmHeight(accountBlockHash)
	if ghErr != nil {
		c.log.Error("GetConfirmHeight failed, error is "+ghErr.Error(), "method", "GetConfirmTimes")
		return 0, ghErr
	}

	if height <= 0 {
		return 0, nil
	}

	return c.GetLatestSnapshotBlock().Height - height + 1, nil
}

func (c *chain) binarySearchBeforeTime(start, end *ledger.SnapshotBlock, blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error) {
	for {
		if end.Height-start.Height <= 1 {
			if start.SnapshotContent == nil {
				var err error
				start.SnapshotContent, err = c.chainDb.Sc.GetSnapshotContent(start.Height)
				if err != nil {
					c.log.Error("GetSnapshotContent failed, error is "+err.Error(), "method", "GetSnapshotBlockBeforeTime")
					return nil, err
				}
			}
			return start, nil
		}

		gap := uint64(end.Timestamp.Sub(*blockCreatedTime).Seconds())
		middle := uint64(0)
		// suppose one snapshot block per second
		if end.Height > gap {
			middle = end.Height - gap
		}
		if middle <= start.Height {
			middle = start.Height + (end.Height-start.Height)/2
		}

		block, err := c.chainDb.Sc.GetSnapshotBlock(middle, false)
		if err != nil {
			c.log.Error("Get try block failed, error is "+err.Error(), "method", "GetSnapshotBlockBeforeTime")
			return nil, err
		}

		prevBlock, err := c.chainDb.Sc.GetSnapshotBlock(middle-1, false)
		if err != nil {
			c.log.Error("Get try block failed, error is "+err.Error(), "method", "GetSnapshotBlockBeforeTime")
			return nil, err
		}

		if block.Timestamp.Before(*blockCreatedTime) {
			start = block
		} else if prevBlock.Timestamp.Before(*blockCreatedTime) {
			start = prevBlock
			end = block
		} else {
			end = prevBlock
		}
	}
}

func (c *chain) GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error) {
	// normal logic
	start := c.GetGenesisSnapshotBlock()
	end := c.GetLatestSnapshotBlock()
	if end.Timestamp.Before(*blockCreatedTime) {
		return end, nil
	}
	if start.Timestamp.After(*blockCreatedTime) {
		return nil, nil
	}

	return c.binarySearchBeforeTime(start, end, blockCreatedTime)
}

func (c *chain) GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error) {
	account, getAccountIdErr := c.chainDb.Account.GetAccountByAddress(address)
	if getAccountIdErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+getAccountIdErr.Error(), "method", "GetConfirmAccountBlock")
		return nil, getAccountIdErr
	}
	if account == nil {
		return nil, nil
	}

	accountBlock, err := c.chainDb.Ac.GetConfirmAccountBlock(snapshotHeight, account.AccountId)
	if err != nil {
		c.log.Error("GetConfirmAccountBlock failed, error is "+err.Error(), "method", "GetConfirmAccountBlock")
		return nil, err
	}

	if accountBlock != nil {
		c.completeBlock(accountBlock, account)
	}

	return accountBlock, nil
}

func (c *chain) getNeedSnapshotMapByDeleteSubLedger(deleteSubLedger map[types.Address][]*ledger.AccountBlock) (map[types.Address]*ledger.AccountBlock, []types.Address, map[types.Address]uint64, error) {
	needAddBlocks := make(map[types.Address]*ledger.AccountBlock)
	var needRemoveAddr []types.Address

	blockHeightMap := make(map[types.Address]uint64)
	for addr, accountBlocks := range deleteSubLedger {
		accountBlock := accountBlocks[0]

		needRemoveAddr = append(needRemoveAddr, addr)
		accountBlockHeight := accountBlock.Height

		if accountBlockHeight > 1 {
			blockHeightMap[addr] = accountBlockHeight - 1
		}
	}

	for addr, blockHeight := range blockHeightMap {
		account, err := c.GetAccount(&addr)
		if err != nil {
			c.log.Error("GetAccount failed, error is "+err.Error(), "method", "getNeedSnapshotMapByDeleteSubLedger")
			return nil, nil, nil, err
		}

		blockHash, blockHashErr := c.chainDb.Ac.GetHashByHeight(account.AccountId, blockHeight)
		if blockHashErr != nil {
			c.log.Error("GetHashByHeight failed, error is "+blockHashErr.Error(), "method", "getNeedSnapshotMapByDeleteSubLedger")
			return nil, nil, nil, err
		}

		if blockHash == nil {
			err := errors.New("blockHash is nil")
			c.log.Error(err.Error(), "method", "getNeedSnapshotMapByDeleteSubLedger")
			return nil, nil, nil, err
		}

		blockMeta, blockMetaErr := c.chainDb.Ac.GetBlockMeta(blockHash)
		if blockMetaErr != nil {
			c.log.Error("GetBlockMeta failed, error is "+blockMetaErr.Error(), "method", "getNeedSnapshotMapByDeleteSubLedger")
			return nil, nil, nil, err
		}

		if blockMeta == nil {
			err := errors.New("blockMeta is nil")
			c.log.Error(err.Error(), "method", "getNeedSnapshotMapByDeleteSubLedger")
			return nil, nil, nil, err
		}

		if blockMeta.SnapshotHeight <= 0 {
			block, blockErr := c.chainDb.Ac.GetBlockByHeight(account.AccountId, blockHeight)
			if blockErr != nil {
				c.log.Error("GetBlockByHeight failed, error is "+blockErr.Error(), "method", "getNeedSnapshotMapByDeleteSubLedger")
				return nil, nil, nil, err
			}

			c.completeBlock(block, account)
			needAddBlocks[account.AccountAddress] = block
		}
	}
	return needAddBlocks, needRemoveAddr, blockHeightMap, nil
}

// Contains to height
func (c *chain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	if toHeight <= 0 || toHeight > c.GetLatestSnapshotBlock().Height {
		return nil, nil, nil
	}

	batch := new(leveldb.Batch)
	snapshotBlocks, accountBlocksMap, err := c.deleteSnapshotBlocksByHeight(batch, toHeight)
	if err != nil {
		c.log.Error("deleteSnapshotBlocksByHeight failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, nil, err
	}

	needAddBlocks, needRemoveAddr, blockHeightMap, err := c.getNeedSnapshotMapByDeleteSubLedger(accountBlocksMap)

	if err != nil {
		c.log.Error("getNeedSnapshotMapByDeleteSubLedger failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksToHeight")
		return nil, nil, err
	}

	needRemoveAddrMap := make(map[types.Address]struct{})
	for _, addr := range needRemoveAddr {
		needRemoveAddrMap[addr] = struct{}{}
	}

	chainRangeSet := c.getChainRangeSet(snapshotBlocks)

	for addr, changeRangeItem := range chainRangeSet {
		min := changeRangeItem[0].Height
		max := changeRangeItem[1].Height

		if blockHeightItem, ok := blockHeightMap[addr]; ok {
			if min > blockHeightItem {
				continue
			}
			max = blockHeightItem
		}

		account, err := c.GetAccount(&addr)
		if err != nil {
			c.log.Error("GetAccount failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksToHeight")
			return nil, nil, err
		}

		for i := min; i <= max; i++ {
			blockHash, blockHashErr := c.chainDb.Ac.GetHashByHeight(account.AccountId, i)
			if blockHashErr != nil {
				c.log.Error("GetHashByHeight failed, error is "+blockHashErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
				return nil, nil, blockHashErr
			}

			// Get be snapshot
			beSnapshot, getBeSnapshotErr := c.chainDb.Ac.GetBeSnapshot(blockHash)

			if getBeSnapshotErr != nil {
				c.log.Error("GetBeSnapshot failed, error is "+getBeSnapshotErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
				return nil, nil, err
			}

			if beSnapshot > 0 {
				c.chainDb.Ac.DeleteBeSnapshot(batch, blockHash)
			}
		}

		if _, ok := needAddBlocks[account.AccountAddress]; !ok {
			needAdd := false
			if _, ok2 := needRemoveAddrMap[account.AccountAddress]; ok2 {
				needAdd = true
			} else if cachedAccountBlock := c.needSnapshotCache.Get(&account.AccountAddress); cachedAccountBlock == nil {
				needAdd = true
			}

			if needAdd {
				var lastBlock *ledger.AccountBlock

				var blockErr error
				lastBlock, blockErr = c.chainDb.Ac.GetBlockByHeight(account.AccountId, max)
				if blockErr != nil {
					c.log.Error("GetBlockByHeight failed, error is "+blockErr.Error(), "method", "DeleteSnapshotBlocksToHeight")
					return nil, nil, err
				}

				c.completeBlock(lastBlock, account)
				needAddBlocks[account.AccountAddress] = lastBlock

			}
		}
	}

	if triggerErr := c.em.triggerDeleteAccountBlocks(batch, accountBlocksMap); triggerErr != nil {
		c.log.Error("c.em.trigger, error is "+triggerErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, triggerErr
	}

	prevSnapshotBlock, prevSnapshotBlockErr := c.chainDb.Sc.GetSnapshotBlock(snapshotBlocks[0].Height-1, true)
	if prevSnapshotBlockErr != nil {
		c.log.Error("GetSnapshotBlock failed, error is "+prevSnapshotBlockErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, prevSnapshotBlockErr
	}

	// Add delete event
	var deleteSbHashList []types.Hash
	var deleteAbHashList []types.Hash

	for _, block := range snapshotBlocks {
		deleteSbHashList = append(deleteSbHashList, block.Hash)
	}
	for _, blocks := range accountBlocksMap {
		for _, block := range blocks {
			deleteAbHashList = append(deleteAbHashList, block.Hash)
		}
	}

	c.chainDb.Be.DeleteSnapshotBlocks(batch, deleteSbHashList)
	c.chainDb.Be.DeleteAccountBlocks(batch, deleteAbHashList)

	// Set needSnapshotCache, first remove
	c.needSnapshotCache.Remove(needRemoveAddr)

	// Set needSnapshotCache, then add
	c.needSnapshotCache.Set(needAddBlocks)

	// write db
	writeErr := c.chainDb.Commit(batch)

	if writeErr != nil {
		c.log.Crit("Write db failed, error is "+writeErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, writeErr
	}

	// Delete cache
	c.stateTriePool.Delete(needRemoveAddr)

	// Set cache
	c.latestSnapshotBlock = prevSnapshotBlock

	// record delete
	c.blackBlock.DeleteSnapshotBlock(snapshotBlocks, accountBlocksMap)

	// Trigger delete snapshot blocks success
	c.em.triggerDeleteSnapshotBlocksSuccess(snapshotBlocks)

	// Trigger delete account blocks success
	c.em.triggerDeleteAccountBlocksSuccess(accountBlocksMap)

	return snapshotBlocks, accountBlocksMap, nil
}

func (c *chain) CheckNeedSnapshotCache(content ledger.SnapshotContent) bool {
	unconfirmSubLedger, err := c.getUnConfirmedSubLedger()
	if err != nil {
		c.log.Error("getUnConfirmedSubLedger failed, error is "+err.Error(), "method", "checkNeedSnapshotCache")
	}
	if len(unconfirmSubLedger) != len(content) {
		return false
	}

	for addr, blocks := range unconfirmSubLedger {
		if block2, ok := content[addr]; !ok || block2.Hash != blocks[0].Hash {
			return false
		}
	}
	return true
}

func (c *chain) deleteSnapshotBlocksByHeight(batch *leveldb.Batch, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	maxAccountId, err := c.chainDb.Account.GetLastAccountId()
	if err != nil {
		c.log.Error("GetLastAccountId failed, error is "+err.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, err
	}

	planToDelete, getPlanErr := c.chainDb.Ac.GetPlanToDelete(maxAccountId, toHeight)
	if getPlanErr != nil {
		c.log.Error("GetPlanToDelete failed, error is "+getPlanErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
	}

	deleteMap, reopenList, getDeleteAndReopenErr := c.chainDb.Ac.GetDeleteMapAndReopenList(planToDelete, c.chainDb.Account.GetAccountByAddress, false, false)
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

	if reopenErr := c.chainDb.Ac.ReopenSendBlocks(batch, reopenList, deleteMap); reopenErr != nil {
		c.log.Error("ReopenSendBlocks failed, error is "+reopenErr.Error(), "method", "DeleteSnapshotBlocksByHeight")
		return nil, nil, reopenErr
	}

	return deleteSnapshotBlocks, subLedger, nil
}

func (c *chain) getChainRangeSet(snapshotBlocks []*ledger.SnapshotBlock) map[types.Address][2]*ledger.HashHeight {
	chainRangeSet := make(map[types.Address][2]*ledger.HashHeight)
	for _, snapshotBlock := range snapshotBlocks {
		for addr, snapshotContent := range snapshotBlock.SnapshotContent {
			height := snapshotContent.Height
			if chainRange := chainRangeSet[addr]; chainRange[0] == nil {
				chainRangeSet[addr] = [2]*ledger.HashHeight{
					{
						Hash:   snapshotContent.Hash,
						Height: snapshotContent.Height,
					}, {
						Hash:   snapshotContent.Hash,
						Height: snapshotContent.Height,
					},
				}
			} else if chainRange[0].Height > height {
				chainRange[0].Hash = snapshotContent.Hash
				chainRange[0].Height = snapshotContent.Height
			} else if chainRange[1].Height < height {
				chainRange[1].Hash = snapshotContent.Hash
				chainRange[1].Height = snapshotContent.Height
			}
		}
	}
	return chainRangeSet
}
