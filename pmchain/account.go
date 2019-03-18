package pmchain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
)

// TODO cache
func (c *chain) IsContractAccount(address *types.Address) (bool, error) {
	result, err := c.stateDB.HasContractMeta(address)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.HasContractMeta failed, error is %s, address is %s", err.Error(), address))
		c.log.Error(cErr.Error(), "method", "IsContractAccount")
		return false, cErr
	}

	return result, nil
}

// TODO cache
func (c *chain) GetAccountId(address *types.Address) (uint64, error) {
	accountId, err := c.indexDB.GetAccountId(address)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountId failed, error is %s, address is %s", err.Error(), address))
		c.log.Error(cErr.Error(), "method", "GetAccountId")
		return 0, cErr
	}

	return accountId, nil
}
