package chain

import (
	"github.com/vitelabs/go-vite/common/types"
)

func (c *chain) HasOnRoadBlocks(addr types.Address) (bool, error) {
	return c.indexDB.HasOnRoad(addr)
}

func (c *chain) GetOnRoadBlocksHashList(addr types.Address, pageNum, num int) ([]types.Hash, error) {
	return c.indexDB.GetOnRoad(addr, pageNum, num)
}
