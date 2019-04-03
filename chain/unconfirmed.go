package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock {
	return c.cache.GetUnconfirmedBlocksByAddress(&addr)
}

func (c *chain) GetContentNeedSnapshot() ledger.SnapshotContent {
	currentUnconfirmedBlocks := c.cache.GetUnconfirmedBlocks()

	needSnapshotBlocks, _ := c.filterCanBeSnapped(currentUnconfirmedBlocks)

	sc := make(ledger.SnapshotContent)

	for i := len(needSnapshotBlocks) - 1; i >= 0; i-- {
		block := needSnapshotBlocks[i]
		if _, ok := sc[block.AccountAddress]; !ok {
			sc[block.AccountAddress] = &ledger.HashHeight{
				Hash:   block.Hash,
				Height: block.Height,
			}
		}
	}

	return sc
}

/*
 * TODO
 * Check quota, consensus, dependencies
 */
func (c *chain) filterCanBeSnapped(blocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, []*ledger.AccountBlock) {
	// checkA()
	//tmpBlocks := make([]*ledger.AccountBlock, 0, len(blocks))
	//if snapshotContent != nil {
	//	for _, block := range blocks {
	//		if hashHeight, ok := snapshotContent[block.AccountAddress]; ok && hashHeight.Height >= block.Height {
	//			continue
	//		}
	//		tmpBlocks = append(tmpBlocks, block)
	//	}
	//}
	//
	//for _, tmpBlock := range tmpBlocks {
	//
	//}
	//quotaCache := map[types.Address]uint64{}
	//
	//for _, block := range blocks {
	//
	//}
	// checkB()
	return blocks, nil
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
