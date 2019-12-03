package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/quota"
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
	// limit account blocks be snapshot less than maxSnapshotLength
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

func (c *chain) filterUnconfirmedBlocks(snapshotBlock *ledger.SnapshotBlock, checkConsensus bool) []*ledger.AccountBlock {
	// get unconfirmed blocks
	blocks := c.cache.GetUnconfirmedBlocks()
	if len(blocks) <= 0 {
		return nil
	}
	// check is active fork point
	if fork.IsActiveForkPoint(snapshotBlock.Height) {
		return blocks
	}

	// get invalidConsensusBlocks
	invalidConsensusBlocks := make(map[types.Hash]*ledger.AccountBlock)

	if checkConsensus {
		invalidBlocks, err := c.filterConsensusFailed(blocks)
		if err != nil {
			c.log.Error(fmt.Sprintf("filterConsensusFailed. Error: %s", err.Error()), "method", "filterInvalidUnconfirmedBlocks")
			// delete all
			return blocks
		}
		// set cap
		invalidConsensusBlocks = make(map[types.Hash]*ledger.AccountBlock, len(invalidBlocks))

		for _, invalidBlock := range invalidBlocks {
			invalidConsensusBlocks[invalidBlock.Hash] = invalidBlock
		}
	}

	// init invalidBlocks
	invalidBlocks := make([]*ledger.AccountBlock, 0)

	// init invalidAddrSet
	invalidAddrSet := make(map[types.Address]struct{})

	// init invalidHashSet
	invalidHashSet := make(map[types.Hash]struct{})

	quotaUsedCache := make(map[types.Address]uint64)
	quotaUnusedCache := make(map[types.Address]uint64)

	for _, block := range blocks {
		valid := true

		addr := block.AccountAddress
		// dependence
		if _, ok := invalidAddrSet[addr]; ok {
			valid = false
		}
		// dependence
		if valid && block.IsReceiveBlock() {
			if block.BlockType == ledger.BlockTypeReceiveError {
				valid = false
			} else if _, ok := invalidHashSet[block.FromBlockHash]; ok {
				valid = false
			}
		}
		// consensus
		if valid && checkConsensus {
			if _, ok := invalidConsensusBlocks[block.Hash]; ok {
				valid = false
				c.log.Info(fmt.Sprintf("will delete block %s, height is %d, addr is %s, invalid consensus", block.Hash, block.Height, block.AccountAddress), "method",
					"filterInvalidUnconfirmedBlocks")
			}
		}

		// quota
		if valid {
			var err error
			// reset quota
			block.Quota, err = quota.CalcBlockQuotaUsed(c, block, snapshotBlock.Height)

			if err != nil {
				c.log.Error(fmt.Sprintf("quota.CalcBlockQuotaUsed failed when filterUnconfirmedBlocks. Error: %s", err), "method", "filterInvalidUnconfirmedBlocks")
				valid = false
			} else if enough, err := c.checkQuota(quotaUnusedCache, quotaUsedCache, block, snapshotBlock.Height); err != nil {
				cErr := errors.New(fmt.Sprintf("c.checkQuota failed, block is %+v. Error: %s", block, err))
				c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
				valid = false
			} else if !enough {
				valid = false
				c.log.Info(fmt.Sprintf("will delete block %s, height is %d, addr is %s, quota is not enough.", block.Hash, block.Height, block.AccountAddress), "method",
					"filterInvalidUnconfirmedBlocks")
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

func (c *chain) checkQuota(quotaUnusedCache map[types.Address]uint64, quotaUsedCache map[types.Address]uint64, block *ledger.AccountBlock, sbHeight uint64) (bool, error) {
	// get quota total
	quotaUnused, ok := quotaUnusedCache[block.AccountAddress]
	if !ok {

		amount, err := c.GetStakeBeneficialAmount(block.AccountAddress)
		if err != nil {
			return false, err
		}

		quotaUnused, err = quota.GetSnapshotCurrentQuota(c, block.AccountAddress, amount, sbHeight)
		if err != nil {
			return false, err
		}
		quotaUnusedCache[block.AccountAddress] = quotaUnused

	}

	quotaUsedCache[block.AccountAddress] += block.Quota

	if quotaUsedCache[block.AccountAddress] > quotaUnused {
		return false, nil
	}

	return true, nil
}

func (c *chain) computeDependencies(accountBlocks []*ledger.AccountBlock) []*ledger.AccountBlock {
	newAccountBlocks := make([]*ledger.AccountBlock, 0, len(accountBlocks))
	newAccountBlocks = append(newAccountBlocks, accountBlocks[0])

	firstAccountBlock := accountBlocks[0]

	addrSet := map[types.Address]struct{}{
		firstAccountBlock.AccountAddress: {},
	}

	hashSet := map[types.Hash]struct{}{
		firstAccountBlock.Hash: {},
	}
	for _, sendBlock := range firstAccountBlock.SendBlockList {
		hashSet[sendBlock.Hash] = struct{}{}
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

func (c *chain) filterConsensusFailed(blocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	contractMetaCache := make(map[types.Address]*ledger.ContractMeta)

	needVerifyBlocks := make(map[types.Gid][]*ledger.AccountBlock)
	for _, block := range blocks {
		addr := block.AccountAddress

		if !types.IsContractAddr(addr) {
			continue
		}

		meta, ok := contractMetaCache[addr]
		if !ok {
			var err error
			meta, err = c.GetContractMeta(addr)
			if err != nil {
				return nil, err
			}
			if meta == nil {
				return nil, errors.New(fmt.Sprintf("%s, meta is nil", addr))
			}

			contractMetaCache[addr] = meta
		}

		gid := meta.Gid
		// add need verify
		needVerifyBlocks[gid] = append(needVerifyBlocks[gid], block)
	}

	return c.consensus.VerifyABsProducer(needVerifyBlocks)
}
