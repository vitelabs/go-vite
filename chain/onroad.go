package chain

import "github.com/vitelabs/go-vite/common/types"

func (c *chain) IsSuccessReceived(addr *types.Address, hash *types.Hash) bool {
	if meta, _ := c.ChainDb().OnRoad.GetMeta(addr, hash); meta != nil {
		return false
	}
	return true
}
