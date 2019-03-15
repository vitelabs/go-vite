package pmchain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) IsAccountBlockExisted(hash *types.Hash) (bool, error) {
	return true, nil
}

func (c *chain) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
	// cache
	if block := c.cache.GetAccountBlockByHeight(addr, height); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetAccountBlockLocation(addr, height)
	if err != nil {
		c.log.Error(fmt.Sprintf("c.indexDB.GetAccountBlockLocation failed, error is %s, address is %s, height is %d",
			err.Error(), addr, height), "method", "GetAccountBlockByHeight")
		return nil, err
	}

	if location == nil {
		return nil, nil
	}

	// query block
	block, err := c.blockDB.GetAccountBlock(location)

	if err != nil {
		c.log.Error(fmt.Sprintf("c.blockDB.GetAccountBlock failed, error is %s,  address is %s, height is %d, location is %s",
			err.Error(), addr, height, location), "method", "GetAccountBlockByHeight")
		return nil, err
	}
	return block, nil
}

func (c *chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {

	return nil, nil
}

// query receive block of send block
func (c *chain) GetReceiveAbBySendAb(sendBlockHash *types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

// is received
func (c *chain) IsReceived(sendBlockHash *types.Hash) (bool, error) {
	return false, nil
}

// high to low, contains the block that has the blockHash
func (c *chain) GetAccountBlocks(blockHash *types.Hash, count uint64) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

// get call depth
func (c *chain) GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error) {
	return 0, nil
}

func (c *chain) GetConfirmedTimes(blockHash *types.Hash) (uint64, error) {
	return 0, nil
}
