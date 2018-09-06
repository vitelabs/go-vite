package chain

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/helper"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type BlockMapQueryParam struct {
	OriginBlockHash *types.Hash
	Count           uint64
	Forward         bool
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

		var startHeight, endHeight, gap = &big.Int{}, &big.Int{}, &big.Int{}
		gap.SetUint64(params.Count)

		if params.Forward {
			startHeight.Set(blockMeta.Height)
			endHeight.Add(startHeight, gap)
			endHeight.Sub(endHeight, big.NewInt(1))
		} else {
			endHeight.Set(blockMeta.Height)
			// Because leveldb iterator range is [a, b)
			startHeight.Sub(endHeight, gap)
			startHeight.Add(startHeight, big.NewInt(1))
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

	endHeight := &big.Int{}
	endHeight.Set(latestBlock.Height)
	endHeight.Sub(endHeight, big.NewInt(int64(index*count)))

	startHeight := &big.Int{}
	startHeight.Set(endHeight)
	startHeight.Sub(startHeight, big.NewInt(int64(num*count)-1))

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
		blockMeta, err := c.chainDb.Ac.GetBlockMeta(block.Hash)
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
