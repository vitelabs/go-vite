package pmchain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) GetUnconfirmedBlocks(addr *types.Address) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *chain) GetContentNeedSnapshot() ledger.SnapshotContent {
	currentUnconfirmedBlocks := c.cache.GetCurrentUnconfirmedBlocks()

	needSnapshotBlocks, _, _ := c.filterCanBeSnapped(currentUnconfirmedBlocks)

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
 * Check quota, consensus, dependencies
 */
func (c *chain) filterCanBeSnapped(blocks []*ledger.AccountBlock) ([]*ledger.AccountBlock, map[types.Address][]*ledger.AccountBlock, []*ledger.AccountBlock) {
	return blocks, nil, nil
}

func blocksToMap(blocks []*ledger.AccountBlock) map[types.Address][]*ledger.AccountBlock {
	blockMap := make(map[types.Address][]*ledger.AccountBlock)
	for _, block := range blocks {
		blockMap[block.AccountAddress] = append(blockMap[block.AccountAddress], block)
	}
	return blockMap
}
