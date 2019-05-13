package api

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"strconv"
)

func NewLedgerApi(vite *vite.Vite) *LedgerApi {
	api := &LedgerApi{
		chain: vite.Chain(),
		//signer:        vite.Signer(),
		log: log15.New("module", "rpc_api/ledger_api"),
	}

	return api
}

type GcStatus struct {
	Code        uint8  `json:"code"`
	Description string `json:"description"`

	ClearedHeight uint64 `json:"clearedHeight"`
	MarkedHeight  uint64 `json:"markedHeight"`
}

type LedgerApi struct {
	chain chain.Chain
	log   log15.Logger
}

func (l LedgerApi) String() string {
	return "LedgerApi"
}

func (l *LedgerApi) ledgerBlockToRpcBlock(block *ledger.AccountBlock) (*AccountBlock, error) {
	return ledgerToRpcBlock(l.chain, block)
}

func (l *LedgerApi) ledgerBlocksToRpcBlocks(list []*ledger.AccountBlock) ([]*AccountBlock, error) {
	var blocks []*AccountBlock
	for _, item := range list {
		rpcBlock, err := l.ledgerBlockToRpcBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, rpcBlock)
	}
	return blocks, nil
}

func (l *LedgerApi) GetBlockByHash(blockHash types.Hash) (*AccountBlock, error) {
	block, getError := l.chain.GetAccountBlockByHash(blockHash)

	if getError != nil {
		l.log.Error("GetAccountBlockByHash failed, error is "+getError.Error(), "method", "GetBlockByHash")

		return nil, getError
	}
	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

func (l *LedgerApi) GetCompleteBlockByHash(blockHash types.Hash) (*AccountBlock, error) {
	block, getError := l.chain.GetCompleteBlockByHash(blockHash)

	if getError != nil {
		l.log.Error("GetCompleteBlockByHash failed, error is "+getError.Error(), "method", "GetCompleteBlockByHash")

		return nil, getError
	}
	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

func (l *LedgerApi) GetBlocksByHash(addr types.Address, originBlockHash *types.Hash, count uint64) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByHash")

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
		l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetBlocksByHash")
		return nil, err
	} else {
		return blocks, nil
	}

}

// in token
func (l *LedgerApi) GetBlocksByHashInToken(addr types.Address, originBlockHash *types.Hash, tokenTypeId types.TokenTypeId, count uint64) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByHashInToken")
	if count == 0 {
		return nil, nil
	}
	plugins := l.chain.Plugins()
	if plugins == nil {
		err := errors.New("config.OpenFilterTokenIndex is false, api can't work")
		return nil, err
	}

	plugin := plugins.GetPlugin("filterToken").(*chain_plugins.FilterToken)

	blocks, err := plugin.GetBlocks(addr, tokenTypeId, originBlockHash, count)
	if err != nil {
		return nil, err
	}

	return l.ledgerBlocksToRpcBlocks(blocks)
}

func (l *LedgerApi) GetVmLogListByHash(logHash types.Hash) (ledger.VmLogList, error) {
	logList, err := l.chain.GetVmLogList(&logHash)
	if err != nil {
		l.log.Error("GetVmLogList failed, error is "+err.Error(), "method", "GetVmLogListByHash")
		return nil, err
	}
	return logList, err
}

func (l *LedgerApi) GetBlocksByHeight(addr types.Address, height uint64, count uint64) ([]*AccountBlock, error) {
	accountBlocks, err := l.chain.GetAccountBlocksByHeight(addr, height, count)
	if err != nil {
		l.log.Error("GetAccountBlocksByHeight failed, error is "+err.Error(), "method", "GetBlocksByHeight")
		return nil, err
	}
	if len(accountBlocks) <= 0 {
		return nil, nil
	}
	return l.ledgerBlocksToRpcBlocks(accountBlocks)
}

func (l *LedgerApi) GetBlockByHeight(addr types.Address, heightStr string) (*AccountBlock, error) {
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		return nil, err
	}

	accountBlock, err := l.chain.GetAccountBlockByHeight(addr, height)
	if err != nil {
		l.log.Error("GetAccountBlockByHeight failed, error is "+err.Error(), "method", "GetBlockByHeight")
		return nil, err
	}

	if accountBlock == nil {
		return nil, nil
	}
	return l.ledgerBlockToRpcBlock(accountBlock)
}

func (l *LedgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByAccAddr")

	height, err := l.chain.GetLatestAccountHeight(addr)
	if err != nil {
		l.log.Error(fmt.Sprintf("GetLatestAccountHeight, addr is %s", addr), "err", err)
		return nil, err
	}

	num := uint64(index * count)
	if height < num {
		return nil, nil
	}

	list, getErr := l.chain.GetAccountBlocksByHeight(addr, height-num, uint64(count))

	if getErr != nil {
		l.log.Info("GetBlocksByAccAddr", "err", getErr)
		return nil, getErr
	}

	if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
		l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetBlocksByAccAddr")
		return nil, err
	} else {
		return blocks, nil
	}
}

func (l *LedgerApi) GetAccountByAccAddr(addr types.Address) (*RpcAccountInfo, error) {
	l.log.Info("GetAccountByAccAddr")

	latestAccountBlock, err := l.chain.GetLatestAccountBlock(addr)
	if err != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	totalNum := uint64(0)
	if latestAccountBlock != nil {
		totalNum = latestAccountBlock.Height
	}

	balanceMap, err := l.chain.GetBalanceMap(addr)
	if err != nil {
		l.log.Error("GetAccountBalance failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	tokenBalanceInfoMap := make(map[types.TokenTypeId]*RpcTokenBalanceInfo)
	for tokenId, amount := range balanceMap {
		token, _ := l.chain.GetTokenInfoById(tokenId)
		if token == nil {
			continue
		}
		tokenBalanceInfoMap[tokenId] = &RpcTokenBalanceInfo{
			TokenInfo:   RawTokenInfoToRpc(token, tokenId),
			TotalAmount: amount.String(),
			Number:      nil,
		}
	}

	rpcAccount := &RpcAccountInfo{
		AccountAddress:      addr,
		TotalNumber:         strconv.FormatUint(totalNum, 10),
		TokenBalanceInfoMap: tokenBalanceInfoMap,
	}

	return rpcAccount, nil
}

func (l *LedgerApi) GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	block, err := l.chain.GetSnapshotBlockByHash(hash)
	if err != nil {
		l.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "GetSnapshotBlockByHash")
	}
	return block, err
}

func (l *LedgerApi) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	block, err := l.chain.GetSnapshotBlockByHeight(height)
	if err != nil {
		l.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "GetSnapshotBlockByHeight")
	}
	return block, err
}

func (l *LedgerApi) GetSnapshotBlocks(height uint64, count int) ([]*ledger.SnapshotBlock, error) {
	blocks, err := l.chain.GetSnapshotBlocksByHeight(height, false, uint64(count))
	if err != nil {
		l.log.Error("GetSnapshotBlocksByHeight failed, error is "+err.Error(), "method", "GetSnapshotBlocks")
	}
	return blocks, nil
}

func (l *LedgerApi) GetChunks(startHeight uint64, endHeight uint64) ([]*ledger.SnapshotChunk, error) {
	chunks, err := l.chain.GetSubLedger(startHeight-1, endHeight)
	if err != nil {
		return nil, err
	}
	if len(chunks) > 0 {
		if chunks[0].SnapshotBlock == nil || chunks[0].SnapshotBlock.Height == startHeight-1 {
			chunks = chunks[1:]
		}
	}
	return chunks, nil

}

func (l *LedgerApi) GetSnapshotChainHeight() string {
	l.log.Info("GetLatestSnapshotChainHeight")
	return strconv.FormatUint(l.chain.GetLatestSnapshotBlock().Height, 10)
}

func (l *LedgerApi) GetLatestSnapshotChainHash() *types.Hash {
	l.log.Info("GetLatestSnapshotChainHash")
	return &l.chain.GetLatestSnapshotBlock().Hash
}

func (l *LedgerApi) GetLatestBlock(addr types.Address) (*AccountBlock, error) {
	l.log.Info("GetLatestBlock")
	block, getError := l.chain.GetLatestAccountBlock(addr)
	if getError != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+getError.Error(), "method", "GetLatestBlock")
		return nil, getError
	}

	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

func (l *LedgerApi) GetVmLogList(blockHash types.Hash) (ledger.VmLogList, error) {
	block, err := l.chain.GetAccountBlockByHash(blockHash)
	if block == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("get block failed")
	}

	return l.chain.GetVmLogList(block.LogHash)
}

func (l *LedgerApi) GetSeed(snapshotHash types.Hash, fromHash types.Hash) (uint64, error) {
	sb, err := l.chain.GetSnapshotBlockByHash(snapshotHash)
	if err != nil {
		return 0, err
	}
	return l.chain.GetSeed(sb, fromHash)
}
