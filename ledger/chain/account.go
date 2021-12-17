package chain

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (c *chain) GetAccountId(address types.Address) (uint64, error) {
	accountId, err := c.indexDB.GetAccountId(&address)

	if err != nil {
		cErr := fmt.Errorf("c.indexDB.GetAccountId failed, error is %s, address is %s", err.Error(), address)
		c.log.Error(cErr.Error(), "method", "GetAccountId")
		return 0, cErr
	}

	return accountId, nil
}

func (c *chain) GetAccountAddress(accountId uint64) (*types.Address, error) {
	addr, err := c.indexDB.GetAccountAddress(accountId)

	if err != nil {
		cErr := fmt.Errorf("c.indexDB.GetAccountAddress failed, error is %s, accountId is %d", err.Error(), accountId)
		c.log.Error(cErr.Error(), "method", "getAccountAddress")
		return nil, cErr
	}

	return addr, nil
}

func (c *chain) IterateAccounts(iterateFunc func(addr types.Address, accountId uint64, err error) bool) {
	c.indexDB.IterateAccounts(iterateFunc)
}

func (c *chain) IterateContracts(iterateFunc func(addr types.Address, meta *ledger.ContractMeta, err error) bool) {
	c.stateDB.IterateContracts(iterateFunc)
}
