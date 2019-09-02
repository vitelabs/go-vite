package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
)

// new api
func (l *LedgerApi) GetAccountBlocks(addr types.Address, originBlockHash *types.Hash, tokenTypeId *types.TokenTypeId, count uint64) ([]*AccountBlock, error) {
	if tokenTypeId == nil {
		if originBlockHash == nil {
			block, err := l.chain.GetLatestAccountBlock(addr)
			if err != nil {
				return nil, err
			}
			if block != nil {
				originBlockHash = &block.Hash
			}
		}

		if originBlockHash == nil {
			return nil, nil
		}

		list, getError := l.chain.GetAccountBlocks(*originBlockHash, count)
		if getError != nil {
			return nil, getError
		}

		if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
			l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetAccountBlocks")
			return nil, err
		} else {
			return blocks, nil
		}
	} else {
		if count == 0 {
			return nil, nil
		}
		plugins := l.chain.Plugins()
		if plugins == nil {
			err := errors.New("config.OpenPlugins is false, api can't work")
			return nil, err
		}

		plugin := plugins.GetPlugin("filterToken").(*chain_plugins.FilterToken)

		blocks, err := plugin.GetBlocks(addr, *tokenTypeId, originBlockHash, count)
		if err != nil {
			return nil, err
		}

		return l.ledgerBlocksToRpcBlocks(blocks)
	}
}

// new api
func (l *LedgerApi) GetAccountBlockByHash(blockHash types.Hash) (*AccountBlock, error) {
	block, getError := l.chain.GetAccountBlockByHash(blockHash)
	if getError != nil {
		l.log.Error("GetAccountBlockByHash failed, error is "+getError.Error(), "method", "GetAccountBlockByHash")

		return nil, getError
	}
	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

// new api
func (l *LedgerApi) GetAccountBlockByHeight(addr types.Address, height interface{}) (*AccountBlock, error) {
	heightUint64, err := parseHeight(height)
	if err != nil {
		return nil, err
	}

	accountBlock, err := l.chain.GetAccountBlockByHeight(addr, heightUint64)
	if err != nil {
		l.log.Error("GetAccountBlockByHeight failed, error is "+err.Error(), "method", "GetAccountBlockByHeight")
		return nil, err
	}

	if accountBlock == nil {
		return nil, nil
	}
	return l.ledgerBlockToRpcBlock(accountBlock)
}

// new api
func (l *LedgerApi) GetAccountBlocksByAddress(addr types.Address, index int, count int) ([]*AccountBlock, error) {
	l.log.Info("GetAccountBlocksByAddress")

	height, err := l.chain.GetLatestAccountHeight(addr)
	if err != nil {
		l.log.Error(fmt.Sprintf("GetLatestAccountHeight, addr is %s", addr), "err", err, "method", "GetAccountBlocksByAddress")
		return nil, err
	}

	num := uint64(index * count)
	if height < num {
		return nil, nil
	}

	list, getErr := l.chain.GetAccountBlocksByHeight(addr, height-num, uint64(count))

	if getErr != nil {
		l.log.Info("GetBlocksByAccAddr", "err", getErr, "method", "GetAccountBlocksByAddress")
		return nil, getErr
	}

	if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
		l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetAccountBlocksByAddress")
		return nil, err
	} else {
		return blocks, nil
	}
}

// new api
func (l *LedgerApi) GetAccountInfoByAddress(addr types.Address) (*AccountInfo, error) {
	l.log.Info("GetAccountInfoByAddress")

	info, err := l.getAccountInfoByAddress(addr)
	if err != nil {
		return nil, err
	}
	return ToAccountInfo(l.chain, info), nil
}

func (l *LedgerApi) getAccountInfoByAddress(addr types.Address) (*ledger.AccountInfo, error) {
	latestAccountBlock, err := l.chain.GetLatestAccountBlock(addr)
	if err != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+err.Error(), "method", "GetAccountInfoByAddress")
		return nil, err
	}

	totalNum := uint64(0)
	if latestAccountBlock != nil {
		totalNum = latestAccountBlock.Height
	}

	balanceMap, err := l.chain.GetBalanceMap(addr)
	if err != nil {
		l.log.Error("GetAccountBalance failed, error is "+err.Error(), "method", "GetAccountInfoByAddress")
		return nil, err
	}

	tokenBalanceInfoMap := make(map[types.TokenTypeId]*ledger.TokenBalanceInfo)
	for tokenId, amount := range balanceMap {
		token, _ := l.chain.GetTokenInfoById(tokenId)
		if token == nil {
			continue
		}
		totalAmount := *big.NewInt(0)
		if amount != nil {
			totalAmount = *amount
		}
		tokenBalanceInfoMap[tokenId] = &ledger.TokenBalanceInfo{
			TotalAmount: totalAmount,
			Number:      0,
		}
	}
	return &ledger.AccountInfo{
		AccountAddress:      addr,
		TotalNumber:         totalNum,
		TokenBalanceInfoMap: tokenBalanceInfoMap,
	}, nil
}

// new api
func (l *LedgerApi) GetLatestSnapshotHash() *types.Hash {
	l.log.Info("GetLatestSnapshotHash")
	return &l.chain.GetLatestSnapshotBlock().Hash
}

// new api
func (l *LedgerApi) GetLatestAccountBlock(addr types.Address) (*AccountBlock, error) {
	l.log.Info("GetLatestAccountBlock")
	block, getError := l.chain.GetLatestAccountBlock(addr)
	if getError != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+getError.Error(), "method", "GetLatestAccountBlock")
		return nil, getError
	}

	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

// new api
func (l *LedgerApi) GetVmLogs(blockHash types.Hash) (ledger.VmLogList, error) {
	block, err := l.chain.GetAccountBlockByHash(blockHash)
	if block == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("get block failed")
	}

	return l.chain.GetVmLogList(block.LogHash)
}

// new api
func (l *LedgerApi) SendRawTransaction(block *AccountBlock) error {

	if block == nil {
		return errors.New("empty block")
	}
	if !checkTxToAddressAvailable(block.ToAddress) {
		return errors.New("ToAddress is invalid")
	}
	lb, err := block.RpcToLedgerBlock()
	if err != nil {
		return err
	}
	if err := checkTokenIdValid(l.chain, &lb.TokenId); err != nil {
		return err
	}
	latestSb := l.chain.GetLatestSnapshotBlock()
	if latestSb == nil {
		return errors.New("failed to get latest snapshotBlock")
	}
	if err := checkSnapshotValid(latestSb); err != nil {
		return err
	}
	if lb.ToAddress == types.AddressDexFund && !dex.VerifyNewOrderPriceForRpc(lb.Data) {
		return dex.InvalidOrderPriceErr
	}

	v := verifier.NewVerifier(nil, verifier.NewAccountVerifier(l.chain, l.vite.Consensus()))
	err = v.VerifyNetAb(lb)
	if err != nil {
		return err
	}
	result, err := v.VerifyRPCAccBlock(lb, latestSb)
	if err != nil {
		return err
	}

	if result != nil {
		return l.vite.Pool().AddDirectAccountBlock(result.AccountBlock.AccountAddress, result)
	} else {
		return errors.New("generator gen an empty block")
	}
}

// new api: ledger_getUnreceivedBlocksByAddress <- onroad_getOnroadBlocksByAddress
func (l *LedgerApi) GetUnreceivedBlocksByAddress(address types.Address, index, count uint64) ([]*AccountBlock, error) {

	log.Info("GetUnreceivedBlocksByAddress", "addr", address, "index", index, "count", count)

	blockList, err := l.chain.GetOnRoadBlocksByAddr(address, int(index), int(count))
	if err != nil {
		return nil, err
	}
	a := make([]*AccountBlock, len(blockList))
	sum := 0
	for _, v := range blockList {
		if v != nil {
			accountBlock, e := ledgerToRpcBlock(l.chain, v)
			if e != nil {
				return nil, e
			}
			a[sum] = accountBlock
			sum++
		}
	}
	return a[:sum], nil
}

// new api: ledger_getUnreceivedTransactionSummaryByAddress <- onroad_getOnroadInfoByAddress
func (l *LedgerApi) GetUnreceivedTransactionSummaryByAddress(address types.Address) (*AccountInfo, error) {

	log.Info("GetUnreceivedTransactionSummaryByAddress", "addr", address)

	info, e := l.chain.GetAccountOnRoadInfo(address)
	if e != nil || info == nil {
		return nil, e
	}
	return ToAccountInfo(l.chain, info), nil
}

// new api: ledger_getUnreceivedBlocksInBatch <- onroad_getOnroadBlocksInBatch
func (l *LedgerApi) GetUnreceivedBlocksInBatch(queryList []PagingQueryBatch) (map[types.Address][]*AccountBlock, error) {
	resultMap := make(map[types.Address][]*AccountBlock)
	for _, q := range queryList {
		if l, ok := resultMap[q.Address]; ok && l != nil {
			continue
		}
		blockList, err := l.GetUnreceivedBlocksByAddress(q.Address, q.PageNumber, q.PageCount)
		if err != nil {
			return nil, err
		}
		if len(blockList) <= 0 {
			continue
		}
		resultMap[q.Address] = blockList
	}
	return resultMap, nil
}

// new api:  ledger_getUnreceivedTransactionSummaryInBatch <-  onroad_getOnroadInfoInBatch
func (l *LedgerApi) GetUnreceivedTransactionSummaryInBatch(addressList []types.Address) ([]*AccountInfo, error) {

	// Remove duplicate
	addrMap := make(map[types.Address]bool, 0)
	for _, v := range addressList {
		addrMap[v] = true
	}

	resultList := make([]*AccountInfo, 0)
	for addr, _ := range addrMap {
		info, err := l.GetUnreceivedTransactionSummaryByAddress(addr)
		if err != nil {
			return nil, err
		}
		if info == nil {
			continue
		}
		resultList = append(resultList, info)
	}
	return resultList, nil
}
