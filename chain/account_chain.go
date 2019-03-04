package chain

import (
	"errors"
	"math/big"

	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm_context"
	"time"
)

type BlockMapQueryParam struct {
	OriginBlockHash *types.Hash
	Count           uint64
	Forward         bool
}

func (c *chain) completeBlock(block *ledger.AccountBlock, account *ledger.Account) {
	block.AccountAddress = account.AccountAddress

	if len(block.PublicKey) <= 0 && len(block.Signature) > 0 {
		block.PublicKey = account.PublicKey
	}
}

func (c *chain) InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error {
	monitorTags := []string{"chain", "InsertAccountBlocks"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	monitor.LogEventNum("chain", "InsertAccountBlocks", len(vmAccountBlocks))

	batch := new(leveldb.Batch)

	trieSaveCallback := make([]func(), 0)
	var account *ledger.Account

	// Write vmContext
	var addBlockHashList []types.Hash
	for _, vmAccountBlock := range vmAccountBlocks {
		accountBlock := vmAccountBlock.AccountBlock

		addBlockHashList = append(addBlockHashList, accountBlock.Hash)

		vmContext := vmAccountBlock.VmContext
		unsavedCache := vmContext.UnsavedCache()
		if unsavedCache != nil {
			// Save trie
			c.saveTrieLock.RLock()
			defer c.saveTrieLock.RUnlock()

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
		}

		if account == nil {
			var getErr error
			if account, getErr = c.chainDb.Account.GetAccountByAddress(&accountBlock.AccountAddress); getErr != nil {
				c.log.Error("GetAccountByAddress failed, error is "+getErr.Error(), "method", "InsertAccountBlocks")
				return getErr
			}

			if account == nil {
				// Create account
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

		// If block is receive block, change status of the send block
		if accountBlock.IsReceiveBlock() {
			sendBlockMeta, getBlockMetaErr := c.chainDb.Ac.GetBlockMeta(&accountBlock.FromBlockHash)
			if getBlockMetaErr != nil {
				c.log.Error("GetBlockMeta failed, error is "+getBlockMetaErr.Error(), "method", "InsertAccountBlocks")
				return getBlockMetaErr
			}

			if sendBlockMeta != nil {
				// Concurrency write block meta

				sendBlockMeta.ReceiveBlockHeights = append(sendBlockMeta.ReceiveBlockHeights, accountBlock.Height)
				saveSendBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.FromBlockHash, sendBlockMeta)
				if saveSendBlockMetaErr != nil {
					c.log.Error("WriteSendBlockMeta failed, error is "+saveSendBlockMetaErr.Error(), "method", "InsertAccountBlocks")
					return saveSendBlockMetaErr
				}
			} else if !c.IsGenesisAccountBlock(accountBlock) {
				err := errors.New(fmt.Sprintf("sendBlockMeta is nil, accountBlock is %+v\n, acccountBlockMeta is %+v\n", accountBlock, accountBlock.Meta))
				c.log.Error(err.Error(), "method", "InsertAccountBlocks")
				return err
			}
		}

		// Save block meta
		blockMeta := &ledger.AccountBlockMeta{
			AccountId:         account.AccountId,
			Height:            accountBlock.Height,
			RefSnapshotHeight: refSnapshotHeight,
		}

		saveBlockMetaErr := c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.Hash, blockMeta)
		if saveBlockMetaErr != nil {
			c.log.Error("WriteBlockMeta failed, error is "+saveBlockMetaErr.Error(), "method", "InsertAccountBlocks")
			return saveBlockMetaErr
		}

		accountBlock.Meta = blockMeta
	}

	// trigger writing event
	if triggerErr := c.em.triggerInsertAccountBlocks(batch, vmAccountBlocks); triggerErr != nil {
		c.log.Error("c.em.trigger, error is "+triggerErr.Error(), "method", "InsertAccountBlocks")
		return triggerErr
	}

	// Set needSnapshotCache, Need first update cache
	if c.needSnapshotCache != nil {
		for _, vmAccountBlock := range vmAccountBlocks {
			err := c.needSnapshotCache.InsertAccountBlock(vmAccountBlock.AccountBlock)
			if err != nil {
				c.log.Error("c.needSnapshotCache.InsertAccountBlock failed, error is "+err.Error(), "method", "InsertAccountBlocks")
				return err
			}
		}
	}

	// Add account block event
	c.chainDb.Be.AddAccountBlocks(batch, addBlockHashList)

	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Crit("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertAccountBlocks")
		return err
	}

	// Set stateTriePool
	lastVmAccountBlock := vmAccountBlocks[len(vmAccountBlocks)-1]

	if lastVmAccountBlock.VmContext.UnsavedCache() != nil {
		c.stateTriePool.Set(&lastVmAccountBlock.AccountBlock.AccountAddress, lastVmAccountBlock.VmContext.UnsavedCache().Trie())
	}

	// After write db
	for _, callback := range trieSaveCallback {
		callback()
	}

	// trigger writing success event
	c.em.triggerInsertAccountBlocksSuccess(vmAccountBlocks)

	// record insert
	c.blackBlock.InsertAccountBlocks(vmAccountBlocks)
	return nil
}

// No block meta
func (c *chain) GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "GetAccountBlocksByHash"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	startHeight := uint64(1)
	if origin != nil {
		blockMeta, gbmErr := c.chainDb.Ac.GetBlockMeta(origin)
		if gbmErr != nil {
			c.log.Error("Query block meta failed. Error is "+gbmErr.Error(), "method", "GetAccountBlocksByHash")
			return nil, gbmErr
		}
		if blockMeta == nil {
			return nil, nil
		}
		startHeight = blockMeta.Height
	} else if !forward {
		account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
		if gaErr != nil {
			c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "GetAccountBlocksByHash")
			return nil, gaErr
		}

		if account == nil {
			return nil, nil
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
	monitorTags := []string{"chain", "GetAccountBlocksByHeight"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	if count <= 0 {
		return nil, nil
	}

	account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
	if gaErr != nil {
		c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "GetAccountBlocksByHeight")
		return nil, gaErr
	}
	if account == nil {
		return nil, nil
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
		c.completeBlock(block, account)
	}
	return blockList, nil
}

// No block meta
func (c *chain) GetAccountBlockMap(queryParams map[types.Address]*BlockMapQueryParam) map[types.Address][]*ledger.AccountBlock {
	monitorTags := []string{"chain", "GetAccountBlockMap"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

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
	monitorTags := []string{"chain", "GetLatestAccountBlock"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		c.log.Error("Query account meta failed. Error is "+err.Error(), "method", "GetLatestAccountBlock")
		return nil, err
	}

	if account == nil {
		return nil, nil
	}

	block, gErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
	if gErr != nil {
		c.log.Error("Query latest block failed. Error is "+gErr.Error(), "method", "GetLatestAccountBlock")

		return nil, err
	}
	if block != nil {
		c.completeBlock(block, account)
	}

	return block, nil
}

func (c *chain) GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error) {
	monitorTags := []string{"chain", "GetAccountBalance"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

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
	monitorTags := []string{"chain", "GetAccountBalanceByTokenId"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

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
	monitorTags := []string{"chain", "GetAccountBlockHashByHeight"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if accountErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "GetAccountBlockHashByHeight")
		return nil, accountErr
	}

	if account == nil {
		return nil, nil
	}

	hash, getHashErr := c.chainDb.Ac.GetHashByHeight(account.AccountId, height)
	if getHashErr != nil {
		c.log.Error("GetHashByHeight failed, error is "+getHashErr.Error(), "method", "GetAccountBlockHashByHeight")
		return nil, getHashErr
	}
	return hash, nil
}

func (c *chain) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "GetAccountBlockByHeight"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		c.log.Error("Query account failed. Error is "+err.Error(), "method", "GetAccountBlockByHeight")

		return nil, err
	}
	if account == nil {
		return nil, nil
	}

	block, err := c.chainDb.Ac.GetBlockByHeight(account.AccountId, height)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}

		c.log.Error("Query block failed. Error is "+err.Error(), "method", "GetAccountBlockByHeight")
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	c.completeBlock(block, account)

	return block, nil
}

// With block meta
func (c *chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "GetAccountBlockByHash"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())
	if blockHash == nil {
		return nil, nil
	}

	block, err := c.chainDb.Ac.GetBlock(blockHash)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}

		c.log.Error("Query block failed. Error is "+err.Error(), "method", "GetAccountBlockByHash")
		return nil, err
	}
	if block == nil {
		return nil, nil
	}

	address, err := c.chainDb.Account.GetAddressById(block.Meta.AccountId)
	if err != nil {
		c.log.Error("Query account id failed. Error is "+err.Error(), "method", "GetAccountBlockByHash")
		return nil, err
	}

	account, err := c.chainDb.Account.GetAccountByAddress(address)
	if err != nil {
		c.log.Error("Query account failed. Error is "+err.Error(), "method", "GetAccountBlockByHash")

		return nil, err
	}

	c.completeBlock(block, account)
	return block, nil
}

func (c *chain) GetAccountBlocksByAddress(addr *types.Address, index, num, count int) ([]*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "GetAccountBlocksByAddress"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	if num == 0 || count == 0 {
		err := errors.New("Num or count can not be 0")
		c.log.Error(err.Error(), "method", "GetAccountBlocksByAddress")

		return nil, err
	}

	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		c.log.Error("Query account meta failed. Error is "+err.Error(), "method", "GetAccountBlocksByAddress")

		return nil, err
	}

	if account == nil {
		return nil, nil
	}

	latestBlock, glErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
	if glErr != nil {

		c.log.Error("Query latest block failed. Error is "+glErr.Error(), "method", "GetAccountBlocksByAddress")
		return nil, glErr
	}
	if latestBlock == nil {
		return nil, nil
	}

	if latestBlock == nil {
		return nil, nil
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
		return nil, err
	}

	// Query block meta list
	for _, block := range blockList {
		c.completeBlock(block, account)
		blockMeta, err := c.chainDb.Ac.GetBlockMeta(&block.Hash)
		if err != nil {
			c.log.Error("Query block meta list failed. Error is "+err.Error(), "method", "GetAccountBlocksByAddress")

			return nil, err
		}
		block.Meta = blockMeta
	}

	return blockList, nil
}

func (c *chain) GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "GetFirstConfirmedAccountBlockBySbHeight"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

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
			return blocks[0], nil
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

		c.completeBlock(block, account)
		return block, nil
	}
}

func (c *chain) GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock {
	monitorTags := []string{"chain", "GetUnConfirmAccountBlocks"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if accountErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "GetUnConfirmAccountBlocks")
		return nil
	}

	if account == nil {
		return nil
	}

	unconfirmedBlocks, err := c.chainDb.Ac.GetUnConfirmAccountBlocks(account.AccountId, 0)
	if err != nil {
		c.log.Error("GetUnConfirmAccountBlocks failed, error is "+err.Error(), "method", "GetUnConfirmAccountBlocks")
		return nil
	}

	return unconfirmedBlocks
}

func (c *chain) DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "DeleteAccountBlocks"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	account, accountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if accountErr != nil {
		c.log.Error("GetAccountByAddress failed, error is "+accountErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, accountErr
	}

	if account == nil {
		return nil, nil
	}

	planToDelete := map[uint64]uint64{account.AccountId: toHeight}

	deleteMap, reopenList, getErr := c.chainDb.Ac.GetDeleteMapAndReopenList(planToDelete, c.chainDb.Account.GetAccountByAddress, true, true)
	if getErr != nil {
		c.log.Error("GetDeleteMapAndReopenList failed, error is "+getErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, getErr
	}

	batch := new(leveldb.Batch)
	deleteAccountBlocks, deleteAccountBlocksErr := c.chainDb.Ac.Delete(batch, deleteMap)
	if len(deleteAccountBlocks) <= 0 {
		return nil, nil
	}
	if deleteAccountBlocksErr != nil {
		c.log.Error("Delete failed, error is "+deleteAccountBlocksErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, deleteAccountBlocksErr
	}

	if reopenErr := c.chainDb.Ac.ReopenSendBlocks(batch, reopenList, deleteMap); reopenErr != nil {
		c.log.Error("ReopenSendBlocks failed, error is "+reopenErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, reopenErr
	}

	subLedger, toSubLedgerErr := c.subLedgerAccountIdToAccountAddress(deleteAccountBlocks)

	if toSubLedgerErr != nil {
		c.log.Error("subLedgerAccountIdToAccountAddress failed, error is "+toSubLedgerErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, toSubLedgerErr
	}

	if triggerErr := c.em.triggerDeleteAccountBlocks(batch, subLedger); triggerErr != nil {
		c.log.Error("c.em.trigger, error is "+triggerErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, triggerErr
	}

	// Write delete blocks event
	var deleteHashList []types.Hash
	var needRemoveAddrList []types.Address
	for addr, accountBlocks := range subLedger {
		needRemoveAddrList = append(needRemoveAddrList, addr)
		for _, block := range accountBlocks {
			deleteHashList = append(deleteHashList, block.Hash)
		}
	}
	c.chainDb.Be.DeleteAccountBlocks(batch, deleteHashList)

	needNotSnapshot := c.calculateNeedNotSnapshot(subLedger)
	c.needSnapshotCache.NotSnapshot(needNotSnapshot)

	writeErr := c.chainDb.Commit(batch)
	if writeErr != nil {
		c.log.Crit("Write db failed, error is "+writeErr.Error(), "method", "DeleteAccountBlocks", "addr", addr, "toHeight", toHeight)
		return nil, writeErr
	}

	// Delete cache
	c.stateTriePool.Delete(needRemoveAddrList)

	c.em.triggerDeleteAccountBlocksSuccess(subLedger)

	// record delete
	c.blackBlock.DeleteAccountBlock(subLedger)

	return subLedger, nil
}
func (c *chain) GetAllLatestAccountBlock() ([]*ledger.AccountBlock, error) {
	monitorTags := []string{"chain", "GetAllLatestAccountBlock"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	maxAccountId, err := c.chainDb.Account.GetLastAccountId()
	if err != nil {
		c.log.Error("GetLastAccountId failed, error is "+err.Error(), "method", "GetAllAccountBlockCount")
		return nil, err
	}

	var allLatestAccountBlock []*ledger.AccountBlock

	for i := uint64(1); i <= maxAccountId; i++ {
		addr, err := c.chainDb.Account.GetAddressById(i)
		if err != nil {
			c.log.Error("GetAddressById failed, error is "+err.Error(), "method", "GetAllAccountBlockCount")
			return nil, err
		}
		if addr == nil {
			err := errors.New("Address is nil")
			c.log.Error(err.Error(), "method", "GetAllAccountBlockCount")
			return nil, err
		}

		account, err := c.chainDb.Account.GetAccountByAddress(addr)
		if err != nil {
			c.log.Error("GetAccountByAddress failed, error is "+err.Error(), "method", "GetAllAccountBlockCount")
			return nil, err
		}
		if account == nil {
			err := errors.New("Account is nil")
			c.log.Error(err.Error(), "method", "GetAllAccountBlockCount")
			return nil, err
		}

		accountBlock, err := c.chainDb.Ac.GetLatestBlock(i)
		if err != nil {
			c.log.Error("GetLatestBlock failed, error is "+err.Error(), "method", "GetAllAccountBlockCount")
			return nil, err
		}
		if accountBlock != nil {
			c.completeBlock(accountBlock, account)
			allLatestAccountBlock = append(allLatestAccountBlock, accountBlock)
		}

	}
	return allLatestAccountBlock, nil
}

// For init need snapshot cache
func (c *chain) GetUnConfirmedSubLedger() (map[types.Address][]*ledger.AccountBlock, error) {
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

// For init need snapshot cache
func (c *chain) GetUnConfirmedPartSubLedger(addrList []types.Address) (map[types.Address][]*ledger.AccountBlock, error) {

	accountIds := make([]uint64, len(addrList))
	for i, addr := range addrList {
		account, err := c.chainDb.Account.GetAccountByAddress(&addr)
		if err != nil {
			c.log.Error("GetAccountByAddress failed, error is "+err.Error(), "method", "getUnConfirmedSubLedger")
			return nil, err
		}
		accountIds[i] = account.AccountId
	}

	subLedger, getErr := c.chainDb.Ac.GetUnConfirmedSubLedgerByAccounts(accountIds)
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
			c.completeBlock(block, account)
		}
	}
	return finalSubLedger, nil
}

func (c *chain) GetAccountBlockMetaByHash(hash *types.Hash) (*ledger.AccountBlockMeta, error) {
	monitorTags := []string{"chain", "GetAccountBlockMetaByHash"}
	defer monitor.LogTimerConsuming(monitorTags, time.Now())

	meta, err := c.chainDb.Ac.GetBlockMeta(hash)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (c *chain) IsAccountBlockExisted(hash types.Hash) (bool, error) {
	isExisted, err := c.chainDb.Ac.IsBlockExisted(hash)
	if err != nil {
		c.log.Error("IsBlockExisted failed, error is "+err.Error(), "method", "IsAccountBlockExisted")
		return false, err
	}
	return isExisted, nil
}

func (c *chain) GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error) {
	blockMeta, err := c.GetAccountBlockMetaByHash(hash)

	if err != nil {
		return nil, err
	}
	if blockMeta == nil {
		return nil, nil
	}

	return blockMeta.ReceiveBlockHeights, nil
}

func (c *chain) IsGenesisAccountBlock(block *ledger.AccountBlock) bool {
	return block.Hash == GenesisMintageBlock.Hash || block.Hash == GenesisMintageSendBlock.Hash || block.Hash == GenesisConsensusGroupBlock.Hash || block.Hash == GenesisRegisterBlock.Hash
}

func (c *chain) GetSendAndReceiveBlocks(accountAddress *types.Address, snapshotBlockHeight uint64) ([]*ledger.AccountBlock, []*ledger.AccountBlock, error) {
	account, err := c.chainDb.Account.GetAccountByAddress(accountAddress)
	if err != nil {
		c.log.Error("GetAccountByAddress failed, error is "+err.Error(), "method", "GetSendAndReceiveBlocks")
		return nil, nil, err
	}

	if account == nil {
		err := errors.New(fmt.Sprintf("account is nil, addr is %s", accountAddress))
		c.log.Error(err.Error(), "method", "GetSendAndReceiveBlocks")
		return nil, nil, err
	}

	sendBlocks, receiveBlocks, err := c.chainDb.Ac.GetSendAndReceiveBlocks(account.AccountId, snapshotBlockHeight)

	if err != nil {
		c.log.Error("GetSendBlocks failed, error is "+err.Error(), "method", "GetSendAndReceiveBlocks")
		return nil, nil, err
	}

	for _, sendBlock := range sendBlocks {
		c.completeBlock(sendBlock, account)
	}

	for _, receiveBlock := range receiveBlocks {
		c.completeBlock(receiveBlock, account)
	}
	return sendBlocks, receiveBlocks, nil
}

func (c *chain) GetOnRoadBlocksBySendAccount(sendAccountAddress *types.Address, snapshotBlockHeight uint64) ([]*ledger.AccountBlock, error) {
	account, err := c.chainDb.Account.GetAccountByAddress(sendAccountAddress)
	if err != nil {
		c.log.Error("GetAccountByAddress failed, error is "+err.Error(), "method", "GetOnRoadBlocksBySendAccount")
		return nil, err
	}

	if account == nil {
		err := errors.New(fmt.Sprintf("account is nil, addr is %s", sendAccountAddress))
		c.log.Error(err.Error(), "method", "GetOnRoadBlocksBySendAccount")
		return nil, err
	}

	sendBlocks, err := c.chainDb.Ac.GetSendBlocks(account.AccountId, snapshotBlockHeight)
	if err != nil {
		c.log.Error("GetOnRoadBlocksBySendAccount failed, error is "+err.Error(), "method", "GetOnRoadBlocksBySendAccount")
		return nil, err
	}

	onRoadBlocks := make([]*ledger.AccountBlock, 0)
	for _, sendBlock := range sendBlocks {
		receiveBlockHeights := sendBlock.Meta.ReceiveBlockHeights

		if len(receiveBlockHeights) > 0 {
			lastReceiveBlockHeight := receiveBlockHeights[len(receiveBlockHeights)-1]
			toAccount, err := c.chainDb.Account.GetAccountByAddress(&sendBlock.ToAddress)
			if err != nil {
				c.log.Error("query to account failed, error is "+err.Error(), "method", "GetOnRoadBlocksBySendAccount")
				return nil, err
			}

			if toAccount == nil {
				err := errors.New("to account is not existed")
				c.log.Error(err.Error(), "method", "GetOnRoadBlocksBySendAccount")
				return nil, err
			}

			lastReceiveBlockHash, err := c.chainDb.Ac.GetHashByHeight(toAccount.AccountId, lastReceiveBlockHeight)
			if err != nil {
				c.log.Error("GetHashByHeight failed, error is "+err.Error(), "method", "GetOnRoadBlocksBySendAccount")
				return nil, err
			}

			if lastReceiveBlockHash == nil {
				err := errors.New("lastReceiveBlockHash is nil")
				c.log.Error(err.Error(), "method", "GetOnRoadBlocksBySendAccount")
				return nil, err
			}

			confirmHeight, err := c.chainDb.Ac.GetConfirmHeight(lastReceiveBlockHash)

			if err != nil {
				c.log.Error("GetConfirmHeight failed, error is "+err.Error(), "method", "GetOnRoadBlocksBySendAccount")
				return nil, err
			}

			if confirmHeight > 0 && confirmHeight <= snapshotBlockHeight {
				continue
			}

		}

		c.completeBlock(sendBlock, account)
		onRoadBlocks = append(onRoadBlocks, sendBlock)

	}

	return onRoadBlocks, nil
}
