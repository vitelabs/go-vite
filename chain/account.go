package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *Chain) GetAccount(address *types.Address) *ledger.Account {
	account, err := c.chainDb.Account.GetAccountByAddress(address)
	if err != nil {
		c.log.Error("Query account failed, error is "+err.Error(), "method", "GetAccount")
	}
	return account
}
