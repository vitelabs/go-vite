package chain

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (c *Chain) GetAccountBlockMap() {

}

func (c *Chain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *Chain) GetAbHashList() {

}

func (c *Chain) GetAccountBalance(addr *types.Address) (map[types.TokenTypeId]*big.Int, error) {
	return nil, nil
}

func (c *Chain) GetAccountBalanceByTokenId(addr *types.Address, tokenId *types.TokenTypeId) (*big.Int, error) {
	return nil, nil
}

func (c *Chain) GetAccountBlocksByAddress(addr *types.Address, index, num, count int) ([]*ledger.AccountBlock, error) {
	account, err := c.chainDb.Account.GetAccountByAddress(addr)
	if err != nil {
		return nil, &types.GetError{
			Code: 1,
			Err:  errors.New("Query account meta failed."),
		}
	}

	blockList, err := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, index, num, count)
	if err != nil {
		return nil, &types.GetError{
			Code: 2,
			Err:  errors.New("Query block list failed."),
		}
	}

	// Query block meta list
	for _, block := range blockList {
		block.PublicKey = account.PublicKey
		blockMeta, err := c.chainDb.Ac.GetBlockMeta(block.Hash)
		if err != nil {
			return nil, &types.GetError{
				Code: 2,
				Err:  errors.New("Query block meta list failed."),
			}
		}
		block.Meta = blockMeta
	}

	return blockList, nil
}
