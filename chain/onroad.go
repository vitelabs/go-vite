package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
)

func (c *chain) HasOnRoadBlocks(address types.Address) (bool, error) {
	result, err := c.indexDB.HasOnRoadBlocks(&address)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.HasOnRoadBlocks failed, error is %s, address is %s", err, address))
		c.log.Error(cErr.Error(), "method", "HasOnRoadBlocks")
		return false, cErr
	}
	return result, nil
}

func (c *chain) GetOnRoadBlocksHashList(address types.Address, pageNum, countPerPage int) ([]types.Hash, error) {
	result, err := c.indexDB.GetOnRoadBlocksHashList(&address, pageNum, countPerPage)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetOnRoadBlocksHashList failed, error is %s, address is %s, pageNum is %d, countPerPage is %d",
			err, address, pageNum, countPerPage))

		c.log.Error(cErr.Error(), "method", "GetOnRoadBlocksHashList")
		return nil, cErr
	}
	return result, nil
}

func (c *chain) DeleteOnRoad(sendBlockHash types.Hash) error {
	//c.flusherMu.RLock()
	//defer c.flusherMu.RUnlock()

	if err := c.indexDB.DeleteOnRoad(sendBlockHash); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.DeleteOnRoad failed, blockHash is %s. Error: %s",
			sendBlockHash, err))

		c.log.Error(cErr.Error(), "method", "GetOnRoadBlocksHashList")
		return cErr
	}
	return nil
}
