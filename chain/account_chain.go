package chain

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type BlockMapQueryParam struct {
	OriginBlockHash *types.Hash
	Count           uint64
	Forward         bool
}

// TODO: WriteBlock
func (c *Chain) InsertAccountBlocks(vmAccountBlocks []*ledger.VmAccountBlock, needBroadCast bool) error {
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
				c.log.Error("SaveTrie failed, error is "+saveTrieErr.Error(), "method", "InsertAccountBlock")
				return saveTrieErr
			} else {
				trieSaveCallback = append(trieSaveCallback, callback)
			}

			// Save log list
			if logList := unsavedCache.LogList(); len(logList) > 0 {
				if err := c.chainDb.Ac.WriteVmLogList(batch, logList); err != nil {
					c.log.Error("WriteVmLogList failed, error is "+err.Error(), "method", "InsertAccountBlock")
					return err
				}
			}

			// Save contract gid list
			if contractGidList := unsavedCache.ContractGidList(); len(contractGidList) > 0 {
				for _, contractGid := range contractGidList {
					if err := c.chainDb.Ac.WriteContractGid(batch, contractGid.Gid(), contractGid.Addr(), contractGid.Open()); err != nil {
						c.log.Error("WriteContractGid failed, error is "+err.Error(), "method", "InsertAccountBlock")
						return err
					}
				}
			}
		}

		if account == nil {
			var getAccountErr error
			if account, getAccountErr = c.chainDb.Account.GetAccountByAddress(&accountBlock.AccountAddress); getAccountErr != nil {
				if getAccountErr == leveldb.ErrNotFound {
					// TODO create account
					// Create account, need lock
				} else {
					c.log.Error("GetAccountByAddress failed, error is "+getAccountErr.Error(), "method", "InsertAccountBlock")
					return getAccountErr
				}
			}
		} else if accountBlock.AccountAddress != account.AccountAddress {
			err := errors.New("AccountAddress is not same")
			c.log.Error("Error is "+err.Error(), "method", "InsertAccountBlock")
			return err
		}

		// Save block
		c.chainDb.Ac.WriteBlock(batch, account.AccountId, accountBlock)

		// Save block meta
		c.chainDb.Ac.WriteBlockMeta(batch, &accountBlock.Hash, accountBlock.Meta)
	}

	// Write db
	if err := c.chainDb.Commit(batch); err != nil {
		c.log.Error("c.chainDb.Commit(batch) failed, error is "+err.Error(), "method", "InsertAccountBlock")
		return err
	}

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

		queryResult[addr] = blockList
	}

	return queryResult
}

func (c *Chain) GetLatestAccountBlock(addr *types.Address) (block *ledger.AccountBlock, err error) {
	defer func() {
		if err != nil {
			c.log.Error(err.Error(), "method", "GetLatestAccountBlock")
		}
	}()

	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		return nil, &types.GetError{
			Code: 1,
			Err:  errors.New("Query account meta failed. Error is " + err.Error()),
		}
	}

	block, gErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
	if gErr != nil {
		return nil, &types.GetError{
			Code: 2,
			Err:  errors.New("Query latest block failed. Error is " + gErr.Error()),
		}
	}
	return block, nil
}

func (c *Chain) GetAbHashList() {

}

func (c *Chain) GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error) {
	return nil, nil
}

func (c *Chain) GetAccountBalanceByTokenId(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error) {
	return nil, nil
}

func (c *Chain) GetAccountBlockByHash(blockHash *types.Hash) (block *ledger.AccountBlock, returnErr error) {
	defer func() {
		if returnErr != nil {
			c.log.Error(returnErr.Error(), "method", "GetAccountBlockByHash")
		}
	}()

	block, err := c.chainDb.Ac.GetBlock(blockHash)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}

		return nil, &types.GetError{
			Code: 1,
			Err:  errors.New("Query block failed. Error is " + err.Error()),
		}
	}

	address, err := c.chainDb.Account.GetAddressById(block.Meta.AccountId)
	if err != nil {
		return nil, &types.GetError{
			Code: 2,
			Err:  errors.New("Query account id failed. Error is " + err.Error()),
		}
	}

	account, err := c.chainDb.Account.GetAccountByAddress(address)
	if err != nil {
		return nil, &types.GetError{
			Code: 3,
			Err:  errors.New("Query account failed. Error is " + err.Error()),
		}
	}

	block.PublicKey = account.PublicKey
	return block, nil
}

func (c *Chain) GetAccountBlocksByAddress(addr *types.Address, index, num, count int) (blocks []*ledger.AccountBlock, err error) {
	defer func() {
		if err != nil {
			c.log.Error(err.Error(), "method", "GetAccountBlocksByAddress")
		}
	}()

	if num == 0 || count == 0 {
		return nil, &types.GetError{
			Code: 1,
			Err:  errors.New("Num or count can not be 0"),
		}
	}

	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		return nil, &types.GetError{
			Code: 2,
			Err:  errors.New("Query account meta failed. Error is " + err.Error()),
		}
	}

	latestBlock, glErr := c.chainDb.Ac.GetLatestBlock(account.AccountId)
	if glErr != nil {
		return nil, &types.GetError{
			Code: 3,
			Err:  errors.New("Query latest block failed. Error is " + glErr.Error()),
		}
	}

	endHeight := latestBlock.Height - uint64(index*count)
	startHeight := endHeight - uint64(num*count) - 1

	blockList, err := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight)

	if err != nil {
		return nil, &types.GetError{
			Code: 4,
			Err:  errors.New("Query block list failed. Error is " + err.Error()),
		}
	}

	helper.ReverseSlice(blockList)

	// Query block meta list
	for _, block := range blockList {
		block.PublicKey = account.PublicKey
		blockMeta, err := c.chainDb.Ac.GetBlockMeta(&block.Hash)
		if err != nil {
			return nil, &types.GetError{
				Code: 5,
				Err:  errors.New("Query block meta list failed. Error is " + err.Error()),
			}
		}
		block.Meta = blockMeta
	}

	return blockList, nil
}
