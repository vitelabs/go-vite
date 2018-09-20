package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *Chain) GetSubLedgerByHeight(startHeight uint64, count int, forward bool) ([]string, [][2]uint64) {
	beginHeight, endHeight := uint64(0), uint64(0)
	if forward {
		beginHeight = startHeight
		endHeight = startHeight + uint64(count) - 1
	} else {
		beginHeight = startHeight - uint64(count) + 1
		endHeight = startHeight
	}

	fileList := c.compressor.Indexer().Get(beginHeight, endHeight)

	var fileNameList []string

	var rangeList [][2]uint64
	for _, fileItem := range fileList {
		fileNameList = append(fileNameList, fileItem.FileName())
		if beginHeight < fileItem.StartHeight() {
			rangeList = append(rangeList, [2]uint64{beginHeight, fileItem.StartHeight()})
		}
		beginHeight = fileItem.EndHeight() + 1
	}

	if len(fileList) > 0 && fileList[len(fileList)-1].EndHeight() < endHeight {
		rangeList = append(rangeList, [2]uint64{fileList[len(fileList)-1].EndHeight() + 1, endHeight})
	}

	return fileNameList, rangeList
}

func (c *Chain) GetSubLedgerByHash(startBlockHash *types.Hash, count int, forward bool) ([]string, [][2]uint64, error) {
	startHeight, err := c.chainDb.Sc.GetSnapshotBlockHeight(startBlockHash)
	if err != nil {
		c.log.Error("GetSnapshotBlockHeight failed, error is "+err.Error(), "method", "GetSubLedgerByHash")
		return nil, nil, err
	}

	if startHeight == 0 {
		c.log.Error("startHeight is 0", "method", "GetSubLedgerByHash")
		return nil, nil, err
	}

	fileNameList, rangeList := c.GetSubLedgerByHeight(startHeight, count, forward)
	return fileNameList, rangeList, nil
}

func (c *Chain) GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	count := toHeight - fromHeight + 1
	snapshotBlocks, err := c.GetSnapshotBlocksByHeight(fromHeight, count, true, true)
	if err != nil {
		return nil, nil, err
	}

	chainRangeSet := c.getChainRangeSet(snapshotBlocks)

	accountChainSubLedger, getErr := c.getChainSet(chainRangeSet)
	if getErr != nil {
		return nil, nil, getErr
	}

	return snapshotBlocks, accountChainSubLedger, nil
}

func (c *Chain) getChainSet(queryParams map[types.Address][2]*ledger.HashHeight) (map[types.Address][]*ledger.AccountBlock, error) {
	queryResult := make(map[types.Address][]*ledger.AccountBlock)
	for addr, params := range queryParams {
		account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
		if gaErr != nil {
			c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "getChainSet")
			return nil, gaErr
		}

		var startHeight, endHeight = params[0].Height, params[1].Height

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
