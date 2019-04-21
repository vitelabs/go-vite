package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) GetAllUnconfirmedBlocks() []*ledger.AccountBlock {
	return c.cache.GetUnconfirmedBlocks()
}

func (c *chain) GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock {
	return c.cache.GetUnconfirmedBlocksByAddress(&addr)
}

const maxSnapshotLength = 40000

func (c *chain) GetContentNeedSnapshot() ledger.SnapshotContent {
	unconfirmedBlocks := c.cache.GetUnconfirmedBlocks()

	sc := make(ledger.SnapshotContent)
	// limit account blocks be snapshot less than 40000
	if len(unconfirmedBlocks) > maxSnapshotLength {
		unconfirmedBlocks = unconfirmedBlocks[:maxSnapshotLength]
	}

	for i := len(unconfirmedBlocks) - 1; i >= 0; i-- {
		block := unconfirmedBlocks[i]
		if _, ok := sc[block.AccountAddress]; !ok {
			sc[block.AccountAddress] = &ledger.HashHeight{
				Hash:   block.Hash,
				Height: block.Height,
			}
		}
	}

	return sc
}

func (c *chain) filterUnconfirmedBlocks(checkConsensus bool) []*ledger.AccountBlock {
	blocks := c.cache.GetUnconfirmedBlocks()
	if len(blocks) <= 0 {
		return nil
	}
	invalidBlocks := make([]*ledger.AccountBlock, 0)

	invalidAddrSet := make(map[types.Address]struct{})
	invalidHashSet := make(map[types.Hash]struct{})

	quotaUsedCache := make(map[types.Address]uint64)
	quotaTotalCache := make(map[types.Address]uint64)

	for _, block := range blocks {

		valid := true

		addr := block.AccountAddress
		// dependence
		if _, ok := invalidAddrSet[addr]; ok {
			valid = false
			// dependence
		} else if block.IsReceiveBlock() {
			if _, ok := invalidHashSet[block.FromBlockHash]; ok {
				valid = false
			}
			// quota
		} else if enough, err := c.checkQuota(quotaTotalCache, quotaUsedCache, block); err != nil {
			cErr := errors.New(fmt.Sprintf("c.checkQuota failed, block is %+v. Error: %s", block, err))
			c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
			return invalidBlocks
			// quota
		} else if !enough {
			valid = false
			// consensus
		} else if checkConsensus {
			if isContract, err := c.IsContractAccount(addr); err != nil {
				cErr := errors.New(fmt.Sprintf("c.IsContractAccount failed, block is %+v. Error: %s", block, err))
				c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
				return invalidBlocks
			} else if isContract {
				ok, err := c.consensus.VerifyAccountProducer(block)
				if err != nil {
					cErr := errors.New(fmt.Sprintf("c.consensus.VerifyAccountProducer failed, block is %+v. Error: %s", block, err))
					c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
					return invalidBlocks
				}
				if !ok {
					valid = false
				}
			}
		}

		if !valid {
			invalidAddrSet[block.AccountAddress] = struct{}{}
			if block.IsSendBlock() {
				invalidHashSet[block.Hash] = struct{}{}
			}
			for _, sendBlock := range block.SendBlockList {
				invalidHashSet[sendBlock.Hash] = struct{}{}
			}
			invalidBlocks = append(invalidBlocks, block)
		}

	}

	return invalidBlocks
}

func (c *chain) checkQuota(quotaTotalCache map[types.Address]uint64, quotaUsedCache map[types.Address]uint64, block *ledger.AccountBlock) (bool, error) {
	quotaTotal, ok := quotaTotalCache[block.AccountAddress]
	if !ok {
		quotaInfo, err := c.GetPledgeQuota(block.AccountAddress)
		if err != nil {
			return false, err
		}
		quotaTotal = quotaInfo.Total()
		quotaUsed, _ := c.cache.GetSnapshotQuotaUsed(&block.AccountAddress)

		quotaTotalCache[block.AccountAddress] = quotaTotal
		quotaUsedCache[block.AccountAddress] = quotaUsed
	}
	quotaUsedCache[block.AccountAddress] += block.Quota

	if quotaUsedCache[block.AccountAddress] > quotaTotal {
		return false, nil
	}

	return true, nil
}

func blocksToMap(blocks []*ledger.AccountBlock) map[types.Address][]*ledger.AccountBlock {
	blockMap := make(map[types.Address][]*ledger.AccountBlock)
	for _, block := range blocks {
		blockMap[block.AccountAddress] = append(blockMap[block.AccountAddress], block)
	}
	return blockMap
}

func (c *chain) computeDependencies(accountBlocks []*ledger.AccountBlock) []*ledger.AccountBlock {
	newAccountBlocks := make([]*ledger.AccountBlock, 0, len(accountBlocks))
	newAccountBlocks = append(newAccountBlocks, accountBlocks[0])

	addrSet := map[types.Address]struct{}{
		accountBlocks[0].AccountAddress: {},
	}

	hashSet := map[types.Hash]struct{}{
		accountBlocks[0].Hash: {},
	}

	length := len(accountBlocks)
	for i := 1; i < length; i++ {

		accountBlock := accountBlocks[i]
		if _, ok := addrSet[accountBlock.AccountAddress]; ok {
			newAccountBlocks = append(newAccountBlocks, accountBlock)
			if accountBlock.IsSendBlock() {
				hashSet[accountBlock.Hash] = struct{}{}
			}
			for _, sendBlock := range accountBlock.SendBlockList {
				hashSet[sendBlock.Hash] = struct{}{}
			}
		} else if accountBlock.IsReceiveBlock() {
			if _, ok := hashSet[accountBlock.FromBlockHash]; ok {
				newAccountBlocks = append(newAccountBlocks, accountBlock)

				addrSet[accountBlock.AccountAddress] = struct{}{}
				for _, sendBlock := range accountBlock.SendBlockList {
					hashSet[sendBlock.Hash] = struct{}{}
				}
			}
		}

	}

	return newAccountBlocks
}
