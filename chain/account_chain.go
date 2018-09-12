package chain

import (
	"errors"
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

func (c *Chain) InsertAccountBlock(accountBlock *ledger.AccountBlock, vmContext *vm_context.VmContext, needBroadCast bool) error {

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
