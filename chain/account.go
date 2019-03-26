package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
)

// TODO cache, unit_test
func (c *chain) IsContractAccount(address types.Address) (bool, error) {
	if ok := types.IsBuiltinContractAddrInUse(address); ok {
		return ok, nil
	}

	result, err := c.stateDB.HasContractMeta(&address)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.HasContractMeta failed, error is %s, address is %s", err.Error(), address))
		c.log.Error(cErr.Error(), "method", "IsContractAccount")
		return false, cErr
	}

	return result, nil
}

func (c *chain) GetAccountId(address types.Address) (uint64, error) {
	accountId, err := c.indexDB.GetAccountId(&address)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountId failed, error is %s, address is %s", err.Error(), address))
		c.log.Error(cErr.Error(), "method", "GetAccountId")
		return 0, cErr
	}

	return accountId, nil
}

func (c *chain) GetAccountAddress(accountId uint64) (*types.Address, error) {
	addr, err := c.indexDB.GetAccountAddress(accountId)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountAddress failed, error is %s, accountId is %d", err.Error(), accountId))
		c.log.Error(cErr.Error(), "method", "getAccountAddress")
		return nil, cErr
	}

	return addr, nil
}
