package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (c *chain) LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error) {
	addrList, err := c.GetContractList(gid)
	if err != nil {
		return nil, err
	}

	onRoadData, err := c.indexDB.Load(addrList)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Load failed, addrList is %+v。 Error: %s", addrList, err))
		c.log.Error(cErr.Error(), "method", "LoadOnRoad")
		return nil, cErr
	}

	return onRoadData, nil

}

func (c *chain) GetOnRoadBlocksByAddr(addr types.Address, pageNum, pageSize int) ([]*ledger.AccountBlock, error) {
	hashList, err := c.indexDB.GetOnRoadHashList(addr, pageNum, pageSize)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.GetOnRoadBlocksByAddr failed, error is %s, address is %s, pageNum is %d, countPerPage is %d",
			err, addr, pageNum, pageSize))
		c.log.Error(cErr.Error(), "method", "GetOnRoadBlocksByAddr")
		return nil, cErr
	}

	blockList := make([]*ledger.AccountBlock, len(hashList))
	count := 0

	for _, v := range hashList {
		b, err := c.GetAccountBlockByHash(v)
		if err != nil {
			return nil, err
		}
		if b == nil {
			c.DeleteOnRoad(addr, v)
			c.log.Error(fmt.Sprintf("block is not exit, hash %s. fix onroad, hash %s is deleted", v, v), "method", "GetOnRoadBlocksByAddr")
			continue
		}
		blockList[count] = b
		count++
	}

	return blockList[:count], nil
}

func (c *chain) DeleteOnRoad(toAddress types.Address, sendBlockHash types.Hash) {
	c.flushMu.RLock()
	defer c.flushMu.RUnlock()
	c.indexDB.DeleteOnRoad(toAddress, sendBlockHash)
}

func (c *chain) GetAccountOnRoadInfo(addr types.Address) (*ledger.AccountInfo, error) {
	if c.plugins == nil {
		return nil, errors.New("plugins-OnRoadInfo's service not provided")
	}
	onRoadInfo, ok := c.plugins.GetPlugin("onRoadInfo").(*chain_plugins.OnRoadInfo)
	if !ok || onRoadInfo == nil {
		return nil, errors.New("plugins-OnRoadInfo's service not provided")
	}
	info, err := onRoadInfo.GetAccountInfo(&addr)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (c *chain) LoadAllOnRoad() (map[types.Address][]types.Hash, error) {
	return c.indexDB.LoadAllHash()
}

func (c *chain) GetOnRoadInfoUnconfirmedHashList(addr types.Address) ([]*types.Hash, error) {
	if c.plugins == nil {
		return nil, errors.New("plugins-OnRoadInfo's service not provided")
	}
	onRoadInfo, ok := c.plugins.GetPlugin("onRoadInfo").(*chain_plugins.OnRoadInfo)
	if !ok || onRoadInfo == nil {
		return nil, errors.New("plugins-OnRoadInfo's service not provided")
	}
	return onRoadInfo.GetOnRoadInfoUnconfirmedHashList(addr)
}

func (c *chain) UpdateOnRoadInfo(addr types.Address, tkId types.TokenTypeId, number uint64, amount big.Int) error {
	if c.plugins == nil {
		return errors.New("plugins-OnRoadInfo's service not provided")
	}
	onRoadInfo, ok := c.plugins.GetPlugin("onRoadInfo").(*chain_plugins.OnRoadInfo)
	if !ok || onRoadInfo == nil {
		return errors.New("plugins-OnRoadInfo's service not provided")
	}
	return onRoadInfo.UpdateOnRoadInfo(addr, tkId, number, amount)
}

func (c *chain) ClearOnRoadUnconfirmedCache(addr types.Address, hashList []*types.Hash) error {
	if c.plugins == nil {
		return errors.New("plugins-OnRoadInfo's service not provided")
	}
	onRoadInfo, ok := c.plugins.GetPlugin("onRoadInfo").(*chain_plugins.OnRoadInfo)
	if !ok || onRoadInfo == nil {
		return errors.New("plugins-OnRoadInfo's service not provided")
	}
	return onRoadInfo.RemoveFromUnconfirmedCache(addr, hashList)
}
