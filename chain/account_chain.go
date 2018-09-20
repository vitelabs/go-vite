package chain

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

type BlockMapQueryParam struct {
	OriginBlockHash *types.Hash
	Count           uint64
	Forward         bool
}

func (c *Chain) InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error {
	batch := new(leveldb.Batch)
	trieSaveCallback := make([]func(), 0)
	var account *ledger.Account

	// Write vmContext
	for _, vmAccountBlock := range vmAccountBlocks {
		accountBlock := vmAccountBlock.AccountBlock
		vmContext := vmAccountBlock.VmContext

		if vmContext != nil {
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

			if contractGidList := unsavedCache.ContractGidList(); len(contractGidList) > 0 {
				for _, contractGid := range contractGidList {
					c.chainDb.Ac.WriteContractGid(batch, contractGid.Gid(), contractGid.Addr())
				}
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

				if err := c.createAccount(batch, accountId, &accountBlock.AccountAddress, accountBlock.PublicKey); err != nil {
					c.log.Error("createAccount failed, error is "+getErr.Error(), "method", "InsertAccountBlocks")
					return err
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

			sendBlockMeta.ReceiveBlockHeight = accountBlock.Height
			saveSendBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.Hash, sendBlockMeta)
			if saveSendBlockMetaErr != nil {
				c.log.Error("WriteSendBlockMeta failed, error is "+saveSendBlockMetaErr.Error(), "method", "InsertAccountBlocks")
				return saveSendBlockMetaErr
			}
		}

		saveBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.Hash, blockMeta)
		if saveBlockMetaErr != nil {
			c.log.Error("WriteBlockMeta failed, error is "+saveBlockMetaErr.Error(), "method", "InsertAccountBlocks")
			return saveBlockMetaErr
		}
	}

	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Error("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertAccountBlocks")
		return err
	}

	lastVmAccountBlock := vmAccountBlocks[len(vmAccountBlocks)-1]

	// Set needSnapshotCache
	c.needSnapshotCache.Add(&account.AccountAddress, lastVmAccountBlock.AccountBlock.Height, &lastVmAccountBlock.AccountBlock.Hash)

	// Set stateTriePool
	c.stateTriePool.Set(&lastVmAccountBlock.AccountBlock.AccountAddress, lastVmAccountBlock.VmContext.UnsavedCache().Trie())

	// After write db
	for _, callback := range trieSaveCallback {
		callback()
	}

	return nil
}

func (c *Chain) GetAccountBlockMap(queryParams map[types.Address]*BlockMapQueryParam) map[types.Address][]*ledger.AccountBlock {
	queryResult := make(map[types.Address][]*ledger.AccountBlock)
	for addr, params := range queryParams {
		account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
		if gaErr != nil {
			c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "GetAccountBlockMap")
			continue
		}

		blockMeta, gbmErr := c.chainDb.Ac.GetBlockMeta(params.OriginBlockHash)
		if gbmErr != nil {
			c.log.Error("Query block meta failed. Error is "+gbmErr.Error(), "method", "GetAccountBlockMap")
			continue
		}

		var startHeight, endHeight = uint64(0), uint64(0)

		if params.Forward {
			startHeight = blockMeta.Height
			endHeight = startHeight + params.Count - 1
		} else {
			endHeight = blockMeta.Height
			startHeight = endHeight - params.Count + 1
		}

		blockList, gbErr := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight)
		if gbErr != nil {
			c.log.Error("Query block failed. Error is "+gbErr.Error(), "method", "GetAccountBlockMap")
			continue
		}

		for _, block := range blockList {
			block.PublicKey = account.PublicKey
		}

		queryResult[addr] = blockList
	}

	return queryResult
}

func (c *Chain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
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
		block.PublicKey = account.PublicKey
	}

	return block, nil
}

func (c *Chain) GetAbHashList(originBlockHash *types.Hash, count, step int, forward bool) ([]*types.Hash, error) {
	block, err := c.GetAccountBlockByHash(originBlockHash)
	if block == nil || err != nil {
		if err != nil {
			c.log.Error("GetAccountBlockByHash failed, error is "+err.Error(), "method", "GetAbHashList")
			return nil, err
		}
		return nil, nil
	}

	account, err := c.chainDb.Account.GetAccountByAddress(&block.AccountAddress)
	if account == nil || err != nil {
		if err != nil {
			c.log.Error("GetAccountByAddress failed, error is "+err.Error(), "method", "GetAbHashList")
			return nil, err
		}
		return nil, nil
	}

	return c.chainDb.Ac.GetAbHashList(account.AccountId, block.Height, count, step, forward), nil
}

func (c *Chain) GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error) {
	trie, err := c.stateTriePool.Get(addr)
	if err != nil {
		c.log.Error("GetTrie failed, error is "+err.Error(), "method", "GetAccountBalanceByTokenId")
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
			c.log.Error("types.BytesToTokenTypeId failed, error is "+err.Error(), "method", "GetAccountBalanceByTokenId")
			return nil, err
		}

		balance := big.NewInt(0)
		balance.SetBytes(value)

		balanceMap[tokenId] = balance
	}

	return balanceMap, nil
}

func (c *Chain) GetAccountBalanceByTokenId(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error) {
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

func (c *Chain) GetAccountBlockHashByHeight(addr *types.Address, height uint64) (*types.Hash, error) {
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

func (c *Chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
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

	block.PublicKey = account.PublicKey
	return block, nil
}

func (c *Chain) GetAccountBlocksByAddress(addr *types.Address, index, num, count int) ([]*ledger.AccountBlock, error) {
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
		startHeight = endHeight - uint64(num*count)
	}

	blockList, err := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight)

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
		block.PublicKey = account.PublicKey
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

// TODO
func (c *Chain) GetFirstUnConfirmAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

// TODO
func (c *Chain) GetFirstUnConfirmAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *Chain) GetUnConfirmAccountBlocks(addr *types.Address) ([]*ledger.AccountBlock, error) {
	account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if accountErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "GetUnConfirmAccountBlocks")
		return nil, &types.GetError{
			Code: 1,
			Err:  accountErr,
		}
	}

	if account != nil {
		return nil, nil
	}

	blocks, unConfirmErr := c.chainDb.Ac.GetUnConfirmAccountBlocks(account.AccountId, 0)
	if unConfirmErr != nil {
		c.log.Error("GetUnConfirmAccountBlocks failed, error is "+unConfirmErr.Error(), "method", "GetUnConfirmAccountBlocks")
		return nil, &types.GetError{
			Code: 2,
			Err:  unConfirmErr,
		}
	}

	for _, block := range blocks {
		block.PublicKey = account.PublicKey
	}

	return blocks, nil
}

// Check一下
func (c *Chain) DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error) {
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

	writeErr := c.chainDb.Commit(batch)
	if writeErr != nil {
		c.log.Error("Write db failed, error is "+writeErr.Error(), "method", "DeleteAccountBlocks")
		return nil, writeErr
	}

	for addr, accountBlocks := range deleteAccountBlocks {
		c.needSnapshotCache.Remove(&addr, accountBlocks[len(accountBlocks)-1].Height)
	}

	return deleteAccountBlocks, nil
}
