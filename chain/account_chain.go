package chain

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

type BlockMapQueryParam struct {
	OriginBlockHash *types.Hash
	Count           uint64
	Forward         bool
}

func (c *chain) InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error {
	batch := new(leveldb.Batch)
	trieSaveCallback := make([]func(), 0)
	var account *ledger.Account

	// Write vmContext
	var publicKey ed25519.PublicKey
	for _, vmAccountBlock := range vmAccountBlocks {
		accountBlock := vmAccountBlock.AccountBlock

		if len(publicKey) > 0 {
			if len(accountBlock.PublicKey) <= 0 {
				accountBlock.PublicKey = publicKey
			}
		} else {
			publicKey = accountBlock.PublicKey
		}

		vmContext := vmAccountBlock.VmContext
		unsavedCache := vmContext.UnsavedCache()
		// Save trie
		if callback, saveTrieErr := unsavedCache.Trie().Save(batch); saveTrieErr != nil {
			c.log.Error("SaveTrie failed, error is "+saveTrieErr.Error(), "method", "InsertAccountBlocks")
			return saveTrieErr
		} else {
			trieSaveCallback = append(trieSaveCallback, callback)
		}

		// Save log list
		if logList := unsavedCache.LogList(); len(logList) > 0 {
			if err := c.chainDb.Ac.WriteVmLogList(batch, logList); err != nil {
				c.log.Error("WriteVmLogList failed, error is "+err.Error(), "method", "InsertAccountBlocks")
				return err
			}
		}

		if account == nil {
			var getErr error
			if account, getErr = c.chainDb.Account.GetAccountByAddress(&accountBlock.AccountAddress); getErr != nil {
				c.log.Error("GetAccountByAddress failed, error is "+getErr.Error(), "method", "InsertAccountBlocks")
				// Create account
				return getErr
			}

			if account == nil {
				c.createAccountLock.Lock()
				defer c.createAccountLock.Unlock()

				accountId, newAccountIdErr := c.newAccountId()
				if newAccountIdErr != nil {
					c.log.Error("newAccountId failed, error is "+newAccountIdErr.Error(), "method", "InsertAccountBlocks")
					return newAccountIdErr
				}

				var caErr error
				if account, caErr = c.createAccount(batch, accountId, &accountBlock.AccountAddress, accountBlock.PublicKey); caErr != nil {
					c.log.Error("createAccount failed, error is "+caErr.Error(), "method", "InsertAccountBlocks")
					return caErr
				}
			}

		} else if accountBlock.AccountAddress != account.AccountAddress {
			err := errors.New("AccountAddress is not same")
			c.log.Error("Error is "+err.Error(), "method", "InsertAccountBlocks")
			return err
		}

		// Save block
		saveBlockErr := c.chainDb.Ac.WriteBlock(batch, account.AccountId, accountBlock)
		if saveBlockErr != nil {
			c.log.Error("WriteBlock failed, error is "+saveBlockErr.Error(), "method", "InsertAccountBlocks")
			return saveBlockErr
		}

		// Save block meta
		refSnapshotHeight, getSnapshotHeightErr := c.chainDb.Sc.GetSnapshotBlockHeight(&accountBlock.SnapshotHash)
		if getSnapshotHeightErr != nil {
			c.log.Error("GetSnapshotBlockHeight failed, error is "+getSnapshotHeightErr.Error(), "method", "InsertAccountBlocks")
			return getSnapshotHeightErr
		}

		blockMeta := &ledger.AccountBlockMeta{
			AccountId:         account.AccountId,
			Height:            accountBlock.Height,
			SnapshotHeight:    0,
			RefSnapshotHeight: refSnapshotHeight,
		}

		if accountBlock.BlockType == ledger.BlockTypeReceive {
			sendBlockMeta, getBlockMetaErr := c.chainDb.Ac.GetBlockMeta(&accountBlock.FromBlockHash)
			if getBlockMetaErr != nil {
				c.log.Error("GetBlockMeta failed, error is "+getBlockMetaErr.Error(), "method", "InsertAccountBlocks")
			}

			if sendBlockMeta != nil {
				sendBlockMeta.ReceiveBlockHeights = append(sendBlockMeta.ReceiveBlockHeights, accountBlock.Height)
				saveSendBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.FromBlockHash, sendBlockMeta)
				if saveSendBlockMetaErr != nil {
					c.log.Error("WriteSendBlockMeta failed, error is "+saveSendBlockMetaErr.Error(), "method", "InsertAccountBlocks")
					return saveSendBlockMetaErr
				}
			}
		}

		saveBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.Hash, blockMeta)
		if saveBlockMetaErr != nil {
			c.log.Error("WriteBlockMeta failed, error is "+saveBlockMetaErr.Error(), "method", "InsertAccountBlocks")
			return saveBlockMetaErr
		}
	}

	if triggerErr := c.em.trigger(InsertAccountBlocksEvent, batch, vmAccountBlocks); triggerErr != nil {
		c.log.Error("c.em.trigger, error is "+triggerErr.Error(), "method", "InsertAccountBlocks")
		return triggerErr
	}
	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Error("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertAccountBlocks")
		return err
	}

	lastVmAccountBlock := vmAccountBlocks[len(vmAccountBlocks)-1]

	// Set needSnapshotCache
	if c.needSnapshotCache != nil {
		c.needSnapshotCache.Add(&account.AccountAddress, lastVmAccountBlock.AccountBlock)
	}

	// Set stateTriePool
	c.stateTriePool.Set(&lastVmAccountBlock.AccountBlock.AccountAddress, lastVmAccountBlock.VmContext.UnsavedCache().Trie())

	// After write db
	for _, callback := range trieSaveCallback {
		callback()
	}

	c.em.trigger(InsertAccountBlocksSuccessEvent, vmAccountBlocks)
	return nil
}

// No block meta
func (c *chain) GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error) {
	startHeight := uint64(1)
	if origin != nil {
		blockMeta, gbmErr := c.chainDb.Ac.GetBlockMeta(origin)
		if gbmErr != nil {
			c.log.Error("Query block meta failed. Error is "+gbmErr.Error(), "method", "GetAccountBlocksByHash")
			return nil, gbmErr
		}
		startHeight = blockMeta.Height
	} else if !forward {
		account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
		if gaErr != nil {
			c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "GetAccountBlocksByHash")
			return nil, gaErr
		}

		block, gbErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
		if gbErr != nil {
			c.log.Error("Query block failed. Error is "+gbErr.Error(), "method", "GetAccountBlocksByHash")
			return nil, gbErr
		}

		if block == nil {
			return nil, nil
		}
		startHeight = block.Height
	}

	return c.GetAccountBlocksByHeight(addr, startHeight, count, forward)
}

// No block meta
func (c *chain) GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error) {
	account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
	if gaErr != nil {
		c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "GetAccountBlocksByHeight")
		return nil, gaErr
	}
	var startHeight, endHeight = uint64(1), uint64(1)

	if forward {
		startHeight = start
		endHeight = startHeight + count - 1

	} else {
		endHeight = start
		if endHeight >= count {
			startHeight = endHeight - count + 1
		}
	}

	blockList, gbErr := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight, forward)
	if gbErr != nil {
		c.log.Error("Query block failed. Error is "+gbErr.Error(), "method", "GetAccountBlocksByHeight")
		return nil, gbErr
	}

	for _, block := range blockList {
		block.AccountAddress = account.AccountAddress
		// Not contract account block
		if len(block.PublicKey) == 0 {
			block.PublicKey = account.PublicKey
		}
	}
	return blockList, nil
}

// No block meta
func (c *chain) GetAccountBlockMap(queryParams map[types.Address]*BlockMapQueryParam) map[types.Address][]*ledger.AccountBlock {
	queryResult := make(map[types.Address][]*ledger.AccountBlock)
	for addr, params := range queryParams {
		blockList, gbErr := c.GetAccountBlocksByHash(addr, params.OriginBlockHash, params.Count, params.Forward)
		if gbErr != nil {
			c.log.Error("Query block failed. Error is "+gbErr.Error(), "method", "GetAccountBlockMap")
			continue
		}

		queryResult[addr] = blockList
	}

	return queryResult
}

func (c *chain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		c.log.Error("Query account meta failed. Error is "+err.Error(), "method", "GetLatestAccountBlock")
		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	if account == nil {
		return nil, nil
	}

	block, gErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
	if gErr != nil {
		c.log.Error("Query latest block failed. Error is "+gErr.Error(), "method", "GetLatestAccountBlock")

		return nil, &types.GetError{
			Code: 2,
			Err:  err,
		}
	}
	if block != nil {
		block.AccountAddress = account.AccountAddress
		// Not contract account block
		if len(block.PublicKey) == 0 {
			block.PublicKey = account.PublicKey
		}
	}

	return block, nil
}

func (c *chain) GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error) {
	trie, err := c.stateTriePool.Get(addr)
	if err != nil {
		c.log.Error("GetTrie failed, error is "+err.Error(), "method", "GetAccountBalance")
		return nil, err
	}

	if trie == nil {
		return nil, nil
	}
	storageIterator := trie.NewIterator(vm_context.STORAGE_KEY_BALANCE)
	balanceMap := make(map[types.TokenTypeId]*big.Int)
	prefixKeyLen := len(vm_context.STORAGE_KEY_BALANCE)
	for {
		key, value, ok := storageIterator.Next()
		if !ok {
			break
		}

		tokenIdBytes := key[prefixKeyLen:]
		tokenId, err := types.BytesToTokenTypeId(tokenIdBytes)
		if err != nil {
			c.log.Error("types.BytesToTokenTypeId failed, error is "+err.Error(), "method", "GetAccountBalance")
			return nil, err
		}

		balance := big.NewInt(0)
		balance.SetBytes(value)

		balanceMap[tokenId] = balance
	}

	return balanceMap, nil
}

func (c *chain) GetAccountBalanceByTokenId(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error) {
	trie, err := c.stateTriePool.Get(addr)
	if err != nil {
		c.log.Error("GetTrie failed, error is "+err.Error(), "method", "GetAccountBalanceByTokenId")
		return nil, err
	}

	balance := big.NewInt(0)
	if trie != nil {
		if value := trie.GetValue(vm_context.BalanceKey(tokenId)); value != nil {
			balance.SetBytes(value)
		}
	}

	return balance, nil
}

func (c *chain) GetAccountBlockHashByHeight(addr *types.Address, height uint64) (*types.Hash, error) {
	account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if accountErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "GetAccountBlockHashByHeight")
		return nil, accountErr
	}

	hash, getHashErr := c.chainDb.Ac.GetHashByHeight(account.AccountId, height)
	if getHashErr != nil {
		c.log.Error("GetHashByHeight failed, error is "+getHashErr.Error(), "method", "GetAccountBlockHashByHeight")
		return nil, getHashErr
	}
	return hash, nil
}

func (c *chain) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		c.log.Error("Query account failed. Error is "+err.Error(), "method", "GetAccountBlockByHeight")

		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	block, err := c.chainDb.Ac.GetBlockByHeight(account.AccountId, height)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}

		c.log.Error("Query block failed. Error is "+err.Error(), "method", "GetAccountBlockByHeight")
		return nil, &types.GetError{
			Code: 2,
			Err:  err,
		}
	}

	block.AccountAddress = account.AccountAddress
	// Not contract account block
	if len(block.PublicKey) == 0 {
		block.PublicKey = account.PublicKey
	}
	return block, nil
}

// With block meta
func (c *chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	block, err := c.chainDb.Ac.GetBlock(blockHash)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}

		c.log.Error("Query block failed. Error is "+err.Error(), "method", "GetAccountBlockByHash")
		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	address, err := c.chainDb.Account.GetAddressById(block.Meta.AccountId)
	if err != nil {
		c.log.Error("Query account id failed. Error is "+err.Error(), "method", "GetAccountBlockByHash")
		return nil, &types.GetError{
			Code: 2,
			Err:  err,
		}
	}

	account, err := c.chainDb.Account.GetAccountByAddress(address)
	if err != nil {
		c.log.Error("Query account failed. Error is "+err.Error(), "method", "GetAccountBlockByHash")

		return nil, &types.GetError{
			Code: 3,
			Err:  err,
		}
	}

	block.AccountAddress = account.AccountAddress
	// Not contract account block
	if len(block.PublicKey) == 0 {
		block.PublicKey = account.PublicKey
	}
	return block, nil
}

func (c *chain) GetAccountBlocksByAddress(addr *types.Address, index, num, count int) ([]*ledger.AccountBlock, error) {
	if num == 0 || count == 0 {
		err := errors.New("Num or count can not be 0")
		c.log.Error(err.Error(), "method", "GetAccountBlocksByAddress")

		return nil, &types.GetError{
			Code: 1,
			Err:  err,
		}
	}

	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		c.log.Error("Query account meta failed. Error is "+err.Error(), "method", "GetAccountBlocksByAddress")

		return nil, &types.GetError{
			Code: 2,
			Err:  err,
		}
	}

	latestBlock, glErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
	if glErr != nil {

		c.log.Error("Query latest block failed. Error is "+glErr.Error(), "method", "GetAccountBlocksByAddress")
		return nil, &types.GetError{
			Code: 3,
			Err:  glErr,
		}
	}

	startHeight, endHeight := uint64(1), uint64(0)
	if latestBlock.Height > uint64(index*count) {
		endHeight = latestBlock.Height - uint64(index*count)
	} else {
		return nil, nil
	}

	if endHeight > uint64(num*count) {
		startHeight = endHeight - uint64(num*count) + 1
	}

	blockList, err := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight, false)

	if err != nil {
		c.log.Error("Query block list failed. Error is "+err.Error(), "method", "GetAccountBlocksByAddress")
		return nil, &types.GetError{
			Code: 4,
			Err:  err,
		}
	}

	helper.ReverseSlice(blockList)

	// Query block meta list
	for _, block := range blockList {
		block.AccountAddress = account.AccountAddress
		// Not contract account block
		if len(block.PublicKey) == 0 {
			block.PublicKey = account.PublicKey
		}
		blockMeta, err := c.chainDb.Ac.GetBlockMeta(&block.Hash)
		if err != nil {
			c.log.Error("Query block meta list failed. Error is "+err.Error(), "method", "GetAccountBlocksByAddress")

			return nil, &types.GetError{
				Code: 5,
				Err:  err,
			}
		}
		block.Meta = blockMeta
	}

	return blockList, nil
}

func (c *chain) GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error) {
	gap := snapshotBlockHeight - c.GetLatestSnapshotBlock().Height
	if gap > 1 {
		// Error
		err := errors.New("the difference in height between snapshotBlockHeight and latestSnapshotBlock.Height is greater than one")
		c.log.Error(err.Error(), "method", "GetFirstConfirmedAccountBlockBySbHeight")
		return nil, err
	} else if gap == 1 {
		// Cache
		blocks := c.GetUnConfirmAccountBlocks(addr)
		if len(blocks) > 0 {
			return blocks[len(blocks)-1], nil
		}
		return nil, nil
	} else {
		// Query db
		snapshotContent, gscErr := c.chainDb.Sc.GetSnapshotContent(snapshotBlockHeight)
		if gscErr != nil {
			c.log.Error("GetSnapshotContent failed, error is "+gscErr.Error(), "method", "GetFirstConfirmedAccountBlockBySbHeight")
			return nil, gscErr
		}

		if snapshotContent == nil {
			c.log.Error("snapshotContent is nil", "method", "GetFirstConfirmedAccountBlockBySbHeight")
			return nil, nil
		}

		snapshotItem := snapshotContent[*addr]
		if snapshotItem == nil {
			return nil, nil
		}

		account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
		if accountErr != nil {
			c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "GetFirstConfirmedAccountBlockBySbHeight")
			return nil, accountErr
		}

		block, unConfirmErr := c.chainDb.Ac.GetFirstConfirmedBlockBeforeOrAtAbHeight(account.AccountId, snapshotItem.Height)

		if unConfirmErr != nil {
			c.log.Error("GetFirstConfirmedBlockBeforeOrAtAbHeight failed, error is "+unConfirmErr.Error(), "method", "GetFirstConfirmedAccountBlockBySbHeight")
			return nil, unConfirmErr
		}

		block.AccountAddress = account.AccountAddress
		// Not contract account block
		if len(block.PublicKey) == 0 {
			block.PublicKey = account.PublicKey
		}
		return block, nil
	}
}

func (c *chain) GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock {
	return c.needSnapshotCache.Get(addr)
}

// Check一下
func (c *chain) DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error) {
	account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if accountErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "DeleteAccountBlocks")
		return nil, &types.GetError{
			Code: 1,
			Err:  accountErr,
		}
	}

	if account == nil {
		return nil, nil
	}

	planToDelete := map[uint64]uint64{account.AccountId: toHeight}

	deleteMap, reopenList, getErr := c.chainDb.Ac.GetDeleteMapAndReopenList(planToDelete, true)
	if getErr != nil {
		c.log.Error("GetDeleteMapAndReopenList failed, error is "+getErr.Error(), "method", "DeleteAccountBlocks")
		return nil, &types.GetError{
			Code: 2,
			Err:  getErr,
		}
	}

	batch := new(leveldb.Batch)
	deleteAccountBlocks, deleteAccountBlocksErr := c.chainDb.Ac.Delete(batch, deleteMap)
	if deleteAccountBlocksErr != nil {
		c.log.Error("Delete failed, error is "+deleteAccountBlocksErr.Error(), "method", "DeleteAccountBlocks")
		return nil, deleteAccountBlocksErr
	}

	reopenErr := c.chainDb.Ac.ReopenSendBlocks(batch, reopenList, deleteMap)
	if reopenErr != nil {
		c.log.Error("ReopenSendBlocks failed, error is "+reopenErr.Error(), "method", "DeleteAccountBlocks")
		return nil, reopenErr
	}

	subLedger, toSubLedgerErr := c.subLedgerAccountIdToAccountAddress(deleteAccountBlocks)

	if toSubLedgerErr != nil {
		c.log.Error("subLedgerAccountIdToAccountAddress failed, error is "+toSubLedgerErr.Error(), "method", "DeleteAccountBlocks")
		return nil, toSubLedgerErr
	}

	if triggerErr := c.em.trigger(DeleteAccountBlocksEvent, subLedger); triggerErr != nil {
		c.log.Error("c.em.trigger, error is "+triggerErr.Error(), "method", "DeleteAccountBlocks")
		return nil, triggerErr
	}

	writeErr := c.chainDb.Commit(batch)
	if writeErr != nil {
		c.log.Error("Write db failed, error is "+writeErr.Error(), "method", "DeleteAccountBlocks")
		return nil, writeErr
	}

	for addr, accountBlocks := range subLedger {
		c.needSnapshotCache.Remove(&addr, accountBlocks[len(accountBlocks)-1].Height)
	}

	c.em.trigger(DeleteAccountBlocksSuccessEvent, deleteAccountBlocks)

	return subLedger, nil
}

// For init need snapshot cache
func (c *chain) getUnConfirmedSubLedger() (map[types.Address][]*ledger.AccountBlock, error) {
	maxAccountId, err := c.chainDb.Account.GetLastAccountId()
	if err != nil {
		c.log.Error("GetLastAccountId failed, error is "+err.Error(), "method", "getUnConfirmedAccountBlocks")
		return nil, err
	}
	subLedger, getErr := c.chainDb.Ac.GetUnConfirmedSubLedger(maxAccountId)
	if getErr != nil {
		c.log.Error("GetUnConfirmedSubLedger failed, error is "+getErr.Error(), "method", "getUnConfirmedSubLedger")
		return nil, getErr
	}

	finalSubLedger, finalErr := c.subLedgerAccountIdToAccountAddress(subLedger)
	if finalErr != nil {
		c.log.Error("subLedgerAccountIdToAccountAddress failed, error is "+getErr.Error(), "method", "getUnConfirmedSubLedger")
		return nil, finalErr
	}

	return finalSubLedger, nil
}

func (c *chain) subLedgerAccountIdToAccountAddress(subLedger map[uint64][]*ledger.AccountBlock) (map[types.Address][]*ledger.AccountBlock, error) {
	finalSubLedger := make(map[types.Address][]*ledger.AccountBlock)
	for accountId, chain := range subLedger {
		address, err := c.chainDb.Account.GetAddressById(accountId)
		if err != nil {
			c.log.Error("Query account id failed. Error is "+err.Error(), "method", "subLedgerAccountIdToAccountAddress")
			return nil, err
		}

		account, err := c.chainDb.Account.GetAccountByAddress(address)
		if err != nil {
			c.log.Error("Query account failed. Error is "+err.Error(), "method", "subLedgerAccountIdToAccountAddress")

			return nil, err
		}

		finalSubLedger[account.AccountAddress] = chain
		for _, block := range finalSubLedger[account.AccountAddress] {
			block.AccountAddress = *address
			if len(block.PublicKey) <= 0 {
				block.PublicKey = account.PublicKey
			}
		}
	}
	return finalSubLedger, nil
}
