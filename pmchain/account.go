package pmchain

import "github.com/vitelabs/go-vite/common/types"

func (c *chain) IsContractAccount(address *types.Address) (bool, error) {
	return false, nil
}
