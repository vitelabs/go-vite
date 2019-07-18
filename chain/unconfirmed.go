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

func (c *chain) filterUnconfirmedBlocks(snapshotBlock *ledger.SnapshotBlock, checkConsensus bool) []*ledger.AccountBlock {
	// get unconfirmed blocks
	blocks := c.cache.GetUnconfirmedBlocks()
	if len(blocks) <= 0 {
		return nil
	}
	// check is fork point
	if fork.IsForkPoint(snapshotBlock.Height) {
		return blocks
	}

	invalidBlocks := make([]*ledger.AccountBlock, 0)

	invalidAddrSet := make(map[types.Address]struct{})
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
		// quota
		if valid {
			var err error
			// reset quota
			block.Quota, err = quota.CalcBlockQuota(c, block, snapshotBlock.Height)

			if err != nil {
				c.log.Error(fmt.Sprintf("quota.CalcBlockQuota failed when filterUnconfirmedBlocks. Error: %s", err), "method", "filterInvalidUnconfirmedBlocks")
				valid = false
			} else if enough, err := c.checkQuota(quotaUnusedCache, quotaUsedCache, block, snapshotBlock.Height); err != nil {
				cErr := errors.New(fmt.Sprintf("c.checkQuota failed, block is %+v. Error: %s", block, err))
				c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
				valid = false
			} else if !enough {
				valid = false
			}
		}
		// consensus
		if valid && checkConsensus {
			if isContract, err := c.IsContractAccount(addr); err != nil {
				cErr := errors.New(fmt.Sprintf("c.IsContractAccount failed, block is %+v. Error: %s", block, err))
				c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
				valid = false
			} else if isContract {
				ok, err := c.consensus.VerifyAccountProducer(block)
				if err != nil {
					cErr := errors.New(fmt.Sprintf("c.consensus.VerifyAccountProducer failed, block is %+v. Error: %s", block, err))
					c.log.Error(cErr.Error(), "method", "filterInvalidUnconfirmedBlocks")
					valid = false
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

func (c *chain) checkQuota(quotaUnusedCache map[types.Address]uint64, quotaUsedCache map[types.Address]uint64, block *ledger.AccountBlock, sbHeight uint64) (bool, error) {
	// get quota total
	quotaUnused, ok := quotaUnusedCache[block.AccountAddress]
	if !ok {

		amount, err := c.GetPledgeBeneficialAmount(block.AccountAddress)
		if err != nil {
			return false, err
		}

		quotaUnused, err = quota.CalcSnapshotCurrentQuota(c, block.AccountAddress, amount, sbHeight)
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
