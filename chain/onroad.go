package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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

func (c *chain) GetAccountOnRoadInfo(addr types.Address) (*ledger.AccountInfo, error) {
	if c.plugins == nil {
		return nil, errors.New("OnRoadInfo-onroad's service not provided.")
	}
	onroad := c.plugins.GetPlugin("onRoadInfo").(*chain_plugins.OnRoadInfo)
	if onroad == nil {
		return nil, errors.New("OnRoadInfo-onroad's service not provided.")
	}
	info, err := onroad.GetAccountInfo(&addr)
	if err != nil {
		return nil, err
	}
	return info, nil
}
