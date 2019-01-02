package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (c *chain) GetSubLedgerByHeight(startHeight uint64, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64) {
	beginHeight, endHeight := uint64(1), uint64(0)

	if forward {
		beginHeight = startHeight
		endHeight = startHeight + count - 1
	} else {
		if startHeight > count {
			beginHeight = startHeight - count + 1
		}
		endHeight = startHeight
	}

	if beginHeight > endHeight {
		return nil, nil
	}

	fileList := c.compressor.Indexer().Get(beginHeight, endHeight)

	var rangeList [][2]uint64

	for _, fileItem := range fileList {
		if beginHeight < fileItem.StartHeight {
			rangeList = append(rangeList, [2]uint64{beginHeight, fileItem.StartHeight})
		}
		beginHeight = fileItem.EndHeight + 1
	}

	if len(fileList) > 0 {
		if fileList[0].StartHeight > beginHeight {
			rangeList = append(rangeList, [2]uint64{beginHeight, fileList[len(fileList)-1].StartHeight - 1})
		}

		if fileList[len(fileList)-1].EndHeight < endHeight {
			rangeList = append(rangeList, [2]uint64{fileList[len(fileList)-1].EndHeight + 1, endHeight})
		}
	} else {
		rangeList = append(rangeList, [2]uint64{beginHeight, endHeight})
	}

	return fileList, rangeList
}

func (c *chain) GetSubLedgerByHash(startBlockHash *types.Hash, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64, error) {
	startHeight, err := c.chainDb.Sc.GetSnapshotBlockHeight(startBlockHash)
	if err != nil {
		c.log.Error("GetSnapshotBlockHeight failed, error is "+err.Error(), "method", "GetSubLedgerByHash")
		return nil, nil, err
	}

	if startHeight == 0 {
		c.log.Error("startHeight is 0", "method", "GetSubLedgerByHash")
		return nil, nil, err
	}

	fileList, rangeList := c.GetSubLedgerByHeight(startHeight, count, forward)
	return fileList, rangeList, nil
}

func (c *chain) GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	count := toHeight - fromHeight + 1
	snapshotBlocks, err := c.GetSnapshotBlocksByHeight(fromHeight, count, true, true)
	if err != nil {
		c.log.Error("GetConfirmSubLedger failed, error is "+err.Error(), "method", "GetConfirmSubLedger")
		return nil, nil, err
	}

	accountChainSubLedger, err := c.GetConfirmSubLedgerBySnapshotBlocks(snapshotBlocks)
	return snapshotBlocks, accountChainSubLedger, err
}

func (c *chain) GetConfirmSubLedgerBySnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) (map[types.Address][]*ledger.AccountBlock, error) {
	chainRangeSet := c.getChainRangeSet(snapshotBlocks)

	accountChainSubLedger, getErr := c.getChainSet(chainRangeSet)
	if getErr != nil {
		c.log.Error("getChainSet failed, error is "+getErr.Error(), "method", "GetConfirmSubLedgerBySnapshotBlocks")
		return nil, getErr
	}

	return accountChainSubLedger, nil
}

func (c *chain) getChainSet(queryParams map[types.Address][2]*ledger.HashHeight) (map[types.Address][]*ledger.AccountBlock, error) {
	queryResult := make(map[types.Address][]*ledger.AccountBlock)
	for addr, params := range queryParams {
		account, gaErr := c.chainDb.Account.GetAccountByAddress(&addr)
		if gaErr != nil {
			c.log.Error("Query account failed. Error is "+gaErr.Error(), "method", "getChainSet")
			return nil, gaErr
		}

		var startHeight, endHeight = params[0].Height, params[1].Height

		blockList, gbErr := c.chainDb.Ac.GetBlockListByAccountId(account.AccountId, startHeight, endHeight, true)

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

		for _, block := range blockList {
			c.completeBlock(block, account)
		}

		queryResult[addr] = blockList
	}

	return queryResult, nil
}
