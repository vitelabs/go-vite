package chain

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) IsGenesisAccountBlock(hash types.Hash) bool {
	return false
}

func (c *chain) IsAccountBlockExisted(hash types.Hash) (bool, error) {
	// cache
	if ok := c.cache.IsAccountBlockExisted(&hash); ok {

		return ok, nil
	}

	// query index
	ok, err := c.indexDB.IsAccountBlockExisted(&hash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.IsAccountBlockExisted failed, error is %s, hash is %s", err, hash))
		return false, cErr
	}

	return ok, nil
}

func (c *chain) GetAccountBlockByHeight(addr types.Address, height uint64) (*ledger.AccountBlock, error) {
	// cache
	if block := c.cache.GetAccountBlockByHeight(&addr, height); block != nil {
		return block, nil
	}

	// query location
	location, err := c.indexDB.GetAccountBlockLocation(&addr, height)
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
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetAccountBlock failed, error is %s,  address is %s, height is %d, location is %+v",
			err.Error(), addr, height, location))
		c.log.Error(cErr.Error(), "method", "GetAccountBlockByHeight")
		return nil, err
	}

	return block, nil
}

func (c *chain) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	var accountBlock *ledger.AccountBlock

	// cache
	if block := c.cache.GetAccountBlockByHash(&blockHash); block != nil {
		if block.IsReceiveBlock() {
			return c.rsBlockToSBlock(block, blockHash), nil
		}
		accountBlock = block
	} else {
		var err error
		// query location
		location, err := c.indexDB.GetAccountBlockLocationByHash(&blockHash)
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
		accountBlock, err = c.blockDB.GetAccountBlock(location)

		if err != nil {
			cErr := errors.New(fmt.Sprintf("c.blockDB.GetAccountBlock failed, error is %s,  hash is %s, location is %+v",
				err.Error(), blockHash, location))
			c.log.Error(cErr.Error(), "method", "GetAccountBlockByHash")
			return nil, cErr
		}
	}

	if accountBlock != nil && accountBlock.IsReceiveBlock() {
		return c.rsBlockToSBlock(accountBlock, blockHash), nil
	}

	return accountBlock, nil
}

// query receive block of send block
// TODO cache
func (c *chain) GetReceiveAbBySendAb(sendBlockHash types.Hash) (*ledger.AccountBlock, error) {
	receiveBlockHash, err := c.indexDB.GetReceivedBySend(&sendBlockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetReceivedBySend failed, hash is %s. Error: %s",
			err.Error(), sendBlockHash))
		c.log.Error(cErr.Error(), "method", "GetReceiveAbBySendAb")
		return nil, cErr
	}

	block, err := c.GetAccountBlockByHash(*receiveBlockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.getAccountBlockByHeight failed, hash is %s. Error: %s"+
			err.Error(), receiveBlockHash))
		c.log.Error(cErr.Error(), "method", "GetReceiveAbBySendAb")
		return nil, cErr
	}
	return block, nil
}

// is received
func (c *chain) IsReceived(sendBlockHash types.Hash) (bool, error) {
	result, err := c.indexDB.IsReceived(&sendBlockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.IsReceived failed, error is %s,  hash is %s",
			err.Error(), sendBlockHash))
		c.log.Error(cErr.Error(), "method", "IsReceived")
		return false, err
	}
	return result, nil
}

// high to low, contains the block that has the blockHash
func (c *chain) GetAccountBlocks(blockHash types.Hash, count uint64) ([]*ledger.AccountBlock, error) {
	addr, locations, heightRange, err := c.indexDB.GetAccountBlockLocationList(&blockHash, count)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountBlockLocationList failed, error is %s,  hash is %s, count is %d",
			err.Error(), blockHash, count))
		c.log.Error(cErr.Error(), "method", "GetAccountBlocks")
		return nil, err
	}
	if len(locations) <= 0 {
		return nil, nil
	}
	return c.getAccountBlocks(*addr, locations, heightRange)
}

// high to low, contains the block that has the blockHash
func (c *chain) GetAccountBlocksByHeight(addr types.Address, height uint64, count uint64) ([]*ledger.AccountBlock, error) {
	locations, heightRange, err := c.indexDB.GetAccountBlockLocationListByHeight(addr, height, count)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetAccountBlockLocationList failed, addr is %s,  height is %s, count is %d.Error: %s",
			addr, height, count, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetAccountBlocksByHeight")
		return nil, err
	}
	if len(locations) <= 0 {
		return nil, nil
	}

	return c.getAccountBlocks(addr, locations, heightRange)
}

// get call depth
func (c *chain) GetCallDepth(sendBlockHash types.Hash) (uint16, error) {
	callDepth, err := c.stateDB.GetCallDepth(&sendBlockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.GetCallDepth failed, sendBlockHash is %s. Error: %s",
			sendBlockHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetConfirmedTimes")
		return 0, cErr
	}
	return callDepth, nil
}

func (c *chain) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	confirmHeight, err := c.indexDB.GetConfirmHeightByHash(&blockHash)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetConfirmHeightByHash failed, blockHash is %s. Error: %s",
			blockHash, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetConfirmedTimes")
		return 0, cErr
	}
	if confirmHeight <= 0 {
		return 0, nil
	}
	latestHeight := c.GetLatestSnapshotBlock().Height
	if latestHeight < confirmHeight {
		return 0, nil
	}
	return latestHeight + 1 - confirmHeight, nil
}

func (c *chain) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	if block := c.cache.GetLatestAccountBlock(&addr); block != nil {
		return block, nil
	}

	height, location, err := c.indexDB.GetLatestAccountBlock(&addr)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetLatestAccountBlock failed, addr is %s. Error: %s",
			addr, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetLatestAccountBlock")
		return nil, cErr
	}
	if height <= 0 {
		return nil, nil
	}

	// cache
	if block := c.cache.GetAccountBlockByHeight(&addr, height); block != nil {
		return block, nil
	}

	// query block
	block, err := c.blockDB.GetAccountBlock(location)

	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.GetAccountBlock failed, address is %s, height is %d, location is %+v. Error: %s, ",
			addr, height, location, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetLatestAccountBlock")
		return nil, err
	}

	return block, nil
}

func (c *chain) GetLatestAccountHeight(addr types.Address) (uint64, error) {
	if block := c.cache.GetLatestAccountBlock(&addr); block != nil {
		return block.Height, nil
	}

	height, _, err := c.indexDB.GetLatestAccountBlock(&addr)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.GetLatestAccountBlock failed, addr is %s. Error: %s",
			addr, err.Error()))
		c.log.Error(cErr.Error(), "method", "GetLatestAccountBlock")
		return 0, cErr
	}
	return height, nil
}

func (c *chain) getAccountBlocks(addr types.Address, locations []*chain_block.Location, heightRange [2]uint64) ([]*ledger.AccountBlock, error) {
	blocks := make([]*ledger.AccountBlock, len(locations))

	startHeight := heightRange[0]
	endHeight := heightRange[1]
	currentHeight := startHeight
	index := 0
	for currentHeight <= endHeight {
		block := c.cache.GetAccountBlockByHeight(&addr, currentHeight)
		if block != nil {
			var err error
			block, err = c.blockDB.GetAccountBlock(locations[index])
			if err != nil {
				cErr := errors.New(fmt.Sprintf("c.blockDB.GetAccountBlock failed, locations is %+v. Error: %s",
					locations[index], err.Error()))
				c.log.Error(cErr.Error(), "method", "getAccountBlocks")
				return nil, cErr
			}
		}

		blocks[index] = block
		index++
		currentHeight++
	}

	return blocks, nil
}

func (c *chain) rsBlockToSBlock(rsBlock *ledger.AccountBlock, blockHash types.Hash) *ledger.AccountBlock {
	if rsBlock.Hash == blockHash {
		return rsBlock
	}
	for i := 0; i < len(rsBlock.SendBlockList); i++ {
		sendBlock := rsBlock.SendBlockList[i]
		if sendBlock.Hash == blockHash {
			return sendBlock
		}
	}
	return nil
}
