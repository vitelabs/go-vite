package pmchain

import "github.com/vitelabs/go-vite/common/types"

func (c *chain) HasOnRoadBlocks(address *types.Address) (bool, error) {
	return false, nil
}

func (c *chain) GetOnRoadBlocksHashList(address *types.Address, pageNum, countPerPage int) ([]*types.Hash, error) {
	return nil, nil
}
