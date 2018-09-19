package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *Chain) GetSubLedgerByHeight(startHeight uint64, count int, forward bool) ([]string, [][2]uint64, error) {
	return nil, nil, nil
}

func (c *Chain) GetSubLedgerByHash(startBlockHash uint64, count int, forward bool) ([]string, [][2]uint64, error) {
	return nil, nil, nil
}

func (c *Chain) GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	count := toHeight - fromHeight + 1
	snapshotBlocks, err := c.GetSnapshotBlocksByHeight(fromHeight, count, true, true)
	if err != nil {
		return nil, nil, err
	}

	chainRangeSet := make(map[types.Address][2]uint64)
	for _, snapshotBlock := range snapshotBlocks {
		for addr, snapshotContent := range snapshotBlock.SnapshotContent {
			height := snapshotContent.AccountBlockHeight
			if chainRange := chainRangeSet[addr]; chainRange[0] == 0 {
				chainRangeSet[addr] = [2]uint64{height, height}
			} else if chainRange[0] > height {
				chainRange[0] = height
			} else if chainRange[1] < height {
				chainRange[1] = height
			}
		}
	}

	accountChainSubLedger, getErr := c.getChainSet(chainRangeSet)
	if getErr != nil {
		return nil, nil, getErr
	}

	return snapshotBlocks, accountChainSubLedger, nil
}

func (c *Chain) getChainSet(queryParams map[types.Address][2]uint64) (map[types.Address][]*ledger.AccountBlock, error) {
	queryResult := make(map[types.Address][]*ledger.AccountBlock)
	for addr, params := range queryParams {
		account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
		if gaErr != nil {
			c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "getChainSet")
			return nil, gaErr
		}

		var startHeight, endHeight = params[0], params[1]

		blockList, gbErr := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight)

		if gbErr != nil {
			c.log.Error("GetBlockListByAccountId failed. Error is "+gbErr.Error(), "method", "getChainSet")
			return nil, gbErr
		}

		otherBlockList, unConfirmErr := c.chainDb.Ac.GetUnConfirmAccountBlocks(account.AccountId, startHeight)
		if unConfirmErr != nil {
			c.log.Error("GetUnConfirmAccountBlocks failed. Error is "+unConfirmErr.Error(), "method", "getChainSet")
			return nil, gbErr
		}

		if otherBlockList != nil {
			blockList = append(otherBlockList, blockList...)
		}

		queryResult[addr] = blockList
	}

	return queryResult, nil
}
