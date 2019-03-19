package pmchain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) IsAccountBlockExisted(hash *types.Hash) (bool, error) {
	// cache
	if ok := c.cache.IsAccountBlockExisted(hash); ok {
		return ok, nil
	}

	// query index
	ok, err := c.indexDB.IsAccountBlockExisted(hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.IsAccountBlockExisted failed, error is %s, hash is %s", err, hash))
		return false, cErr
	}

	return ok, nil
}

func (c *chain) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
	// cache
	if block := c.cache.GetAccountBlockByHeight(addr, height); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetAccountBlockLocation(addr, height)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountBlockLocation failed, error is %s, address is %s, height is %d",
			err.Error(), addr, height))
		c.log.Error(cErr.Error(), "method", "GetAccountBlockByHeight")
		return nil, err
	}

	if location == nil {
		return nil, nil
	}

	// query block
	block, err := c.blockDB.GetAccountBlock(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetAccountBlock failed, error is %s,  address is %s, height is %d, location is %s",
			err.Error(), addr, height, location))
		c.log.Error(cErr.Error(), "method", "GetAccountBlockByHeight")
		return nil, err
	}
	return block, nil
}

func (c *chain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	// cache
	if block := c.cache.GetAccountBlockByHash(blockHash); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetSnapshotBlockLocationByHash(blockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountBlockLocation failed, error is %s,  hash is %s",
			err.Error(), blockHash))
		c.log.Error(cErr.Error(), "method", "GetAccountBlockByHash")
		return nil, err
	}

	if location == nil {
		return nil, nil
	}

	// query block
	block, err := c.blockDB.GetAccountBlock(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetAccountBlock failed, error is %s,  hash is %s, location is %+v",
			err.Error(), blockHash, location))
		c.log.Error(cErr.Error(), "method", "GetAccountBlockByHash")
		return nil, cErr
	}
	return block, nil
}

// query receive block of send block
// TODO cache
func (c *chain) GetReceiveAbBySendAb(sendBlockHash *types.Hash) (*ledger.AccountBlock, error) {
	receiveAccountId, receiveHeight, err := c.indexDB.GetReceivedBySend(sendBlockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetReceivedBySend failed, error is %s,  hash is %s",
			err.Error(), sendBlockHash))
		c.log.Error(cErr.Error(), "method", "GetReceiveAbBySendAb")
	}

	block, err := c.getAccountBlockByHeight(receiveAccountId, receiveHeight)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getAccountBlockByHeight failed, error is %s,  hash is %s, "+
			"receiveAccountId is %d, receiveHeight is %d",
			err.Error(), sendBlockHash, receiveAccountId, receiveHeight))
		c.log.Error(cErr.Error(), "method", "GetReceiveAbBySendAb")
	}
	return block, nil
}

// is received
func (c *chain) IsReceived(sendBlockHash *types.Hash) (bool, error) {
	result, err := c.indexDB.IsReceived(sendBlockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.IsReceived failed, error is %s,  hash is %s",
			err.Error(), sendBlockHash))
		c.log.Error(cErr.Error(), "method", "IsReceived")
		return false, err
	}
	return result, nil
}

// high to low, contains the block that has the blockHash
func (c *chain) GetAccountBlocks(blockHash *types.Hash, count uint64) ([]*ledger.AccountBlock, error) {
	//locations, accountId, heightRange, err := c.indexDB.GetAccountBlockLocationList(blockHash, count)
	//
	//if err != nil {
	//	cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountBlockLocationList failed, error is %s,  hash is %s, count is %d",
	//		err.Error(), blockHash, count))
	//	c.log.Error(cErr.Error(), "method", "GetAccountBlocks")
	//	return nil, err
	//}
	//if len(locations) <= 0 {
	//	return nil, nil
	//}
	//
	//addr, err := c.GetAccount(accountId)
	//if err != nil {
	//	cErr := errors.New(fmt.Sprintf("c.GetAccountId failed, error is %s,  accountId is %d",
	//		err.Error(), accountId))
	//	c.log.Error(cErr.Error(), "method", "GetAccountBlocks")
	//	return nil, err
	//}
	//blocks := make([]*ledger.AccountBlock, len(locations))
	//
	//startHeight := heightRange[0]
	//endHeight := heightRange[1]
	//currentHeight := startHeight
	//index := 0
	//for currentHeight <= endHeight {
	//
	//	block := c.cache.GetAccountBlockByHeight(addr, currentHeight)
	//	if block != nil {
	//
	//	}
	//	blocks[index] = block
	//	index++
	//	currentHeight++
	//}
	//
	return nil, nil
}

// get call depth
func (c *chain) GetCallDepth(sendBlock *ledger.AccountBlock) (uint64, error) {
	return 0, nil
}

func (c *chain) GetConfirmedTimes(blockHash *types.Hash) (uint64, error) {

	return 0, nil
}

func (c *chain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *chain) getAccountBlockByHeight(accountId, height uint64) (*ledger.AccountBlock, error) {
	return nil, nil
}
