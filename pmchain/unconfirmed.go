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
	sc := make(ledger.SnapshotContent)

	for _, block := range currentUnconfirmedBlocks {
		if c.checkCanBeSnapped(block) {
			sc[block.AccountAddress] = &ledger.HashHeight{
				Hash:   block.Hash,
				Height: block.Height,
			}
		}
	}
	return sc
}

func (c *chain) checkCanBeSnapped(block *ledger.AccountBlock) bool {
	return true
}
