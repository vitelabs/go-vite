package api

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
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

func (l *LedgerApi) ledgerChunksToRpcChunks(list []*ledger.SnapshotChunk) ([]*SnapshotChunk, error) {
	chunks := make([]*SnapshotChunk, 0, len(list))
	for _, item := range list {
		sb, err := l.ledgerSnapshotBlockToRpcBlock(item.SnapshotBlock)
		if err != nil {
			return nil, err
		}

		chunks = append(chunks, &SnapshotChunk{
			AccountBlocks: item.AccountBlocks,
			SnapshotBlock: sb,
		})
	}
	return chunks, nil
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

func (l *LedgerApi) ledgerSnapshotBlockToRpcBlock(block *ledger.SnapshotBlock) (*SnapshotBlock, error) {
	return ledgerSnapshotBlockToRpcBlock(block)
}

func (l *LedgerApi) ledgerSnapshotBlocksToRpcBlocks(list []*ledger.SnapshotBlock) ([]*SnapshotBlock, error) {
	var blocks []*SnapshotBlock
	for _, item := range list {
		rpcBlock, err := l.ledgerSnapshotBlockToRpcBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, rpcBlock)
	}

	return blocks, nil
}

func (l *LedgerApi) GetRawBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return l.chain.GetAccountBlockByHash(blockHash)
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
		err := errors.New("config.OpenPlugins is false, api can't work")
		return nil, err
	}

	plugin := plugins.GetPlugin("filterToken").(*chain_plugins.FilterToken)

	blocks, err := plugin.GetBlocks(addr, tokenTypeId, originBlockHash, count)
	if err != nil {
		return nil, err
	}

	return l.ledgerBlocksToRpcBlocks(blocks)
}

func (l *LedgerApi) GetSnapshotBlockBeforeTime(timestamp int64) (*SnapshotBlock, error) {
	time := time.Unix(timestamp, 0)
	sbHeader, err := l.chain.GetSnapshotHeaderBeforeTime(&time)
	if err != nil || sbHeader == nil {
		return nil, err
	}

	sb, err := l.chain.GetSnapshotBlockByHash(sbHeader.Hash)
	if err != nil {
		return nil, err
	}

	return l.ledgerSnapshotBlockToRpcBlock(sb)
}

func (l *LedgerApi) GetVmLogListByHash(logHash types.Hash) (ledger.VmLogList, error) {
	logList, err := l.chain.GetVmLogList(&logHash)
	if err != nil {
		l.log.Error("GetVmLogList failed, error is "+err.Error(), "method", "GetVmLogListByHash")
		return nil, err
	}
	return logList, err
}

func (l *LedgerApi) GetBlocksByHeight(addr types.Address, height interface{}, count uint64) ([]*AccountBlock, error) {
	heightUint64, err := parseHeight(height)
	if err != nil {
		return nil, err
	}

	accountBlocks, err := l.chain.GetAccountBlocksByHeight(addr, heightUint64, count)
	if err != nil {
		l.log.Error("GetAccountBlocksByHeight failed, error is "+err.Error(), "method", "GetBlocksByHeight")
		return nil, err
	}
	if len(accountBlocks) <= 0 {
		return nil, nil
	}
	return l.ledgerBlocksToRpcBlocks(accountBlocks)
}

func (l *LedgerApi) GetBlockByHeight(addr types.Address, height interface{}) (*AccountBlock, error) {
	heightUint64, err := parseHeight(height)
	if err != nil {
		return nil, err
	}

	accountBlock, err := l.chain.GetAccountBlockByHeight(addr, heightUint64)
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

func (l *LedgerApi) GetSnapshotBlockByHash(hash types.Hash) (*SnapshotBlock, error) {
	block, err := l.chain.GetSnapshotBlockByHash(hash)
	if err != nil {
		l.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "GetSnapshotBlockByHash")
		return nil, err
	}
	return l.ledgerSnapshotBlockToRpcBlock(block)
}

func (l *LedgerApi) GetSnapshotBlockByHeight(height interface{}) (*SnapshotBlock, error) {
	heightUint64, err := parseHeight(height)
	if err != nil {
		return nil, err
	}

	block, err := l.chain.GetSnapshotBlockByHeight(heightUint64)
	if err != nil {
		l.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "GetSnapshotBlockByHeight")
		return nil, err
	}
	return l.ledgerSnapshotBlockToRpcBlock(block)
}

func (l *LedgerApi) GetSnapshotBlocks(height interface{}, count int) ([]*SnapshotBlock, error) {
	heightUint64, err := parseHeight(height)
	if err != nil {
		return nil, err
	}

	blocks, err := l.chain.GetSnapshotBlocksByHeight(heightUint64, false, uint64(count))
	if err != nil {
		l.log.Error("GetSnapshotBlocksByHeight failed, error is "+err.Error(), "method", "GetSnapshotBlocks")
		return nil, err
	}
	return l.ledgerSnapshotBlocksToRpcBlocks(blocks)
}

func (l *LedgerApi) GetChunks(startHeight interface{}, endHeight interface{}) ([]*SnapshotChunk, error) {
	startHeightUint64, err := parseHeight(startHeight)
	if err != nil {
		return nil, err
	}

	endHeightUint64, err := parseHeight(endHeight)
	if err != nil {
		return nil, err
	}

	chunks, err := l.chain.GetSubLedger(startHeightUint64-1, endHeightUint64)
	if err != nil {
		return nil, err
	}
	if len(chunks) > 0 {
		if chunks[0].SnapshotBlock == nil || chunks[0].SnapshotBlock.Height == startHeightUint64-1 {
			chunks = chunks[1:]
		}
	}

	return l.ledgerChunksToRpcChunks(chunks)

}

func (l *LedgerApi) GetSnapshotChainHeight() string {
	l.log.Info("GetLatestSnapshotChainHeight")
	return strconv.FormatUint(l.chain.GetLatestSnapshotBlock().Height, 10)
}

func (l *LedgerApi) GetLatestSnapshotChainHash() *types.Hash {
	l.log.Info("GetLatestSnapshotChainHash")
	return &l.chain.GetLatestSnapshotBlock().Hash
}

func (l *LedgerApi) GetLatestSnapshotBlock() (*SnapshotBlock, error) {
	block := l.chain.GetLatestSnapshotBlock()
	return l.ledgerSnapshotBlockToRpcBlock(block)
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

func (l *LedgerApi) GetChainStatus() []interfaces.DBStatus {
	return l.chain.GetStatus()
}

func (l *LedgerApi) GetAllUnconfirmedBlocks() []*ledger.AccountBlock {
	return l.chain.GetAllUnconfirmedBlocks()
}

func (l *LedgerApi) GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock {
	return l.chain.GetUnconfirmedBlocks(addr)
}

type GetBalancesRes map[types.Address]map[types.TokenTypeId]*big.Int

func (l *LedgerApi) GetConfirmedBalances(snapshotHash types.Hash, addrList []types.Address, tokenIds []types.TokenTypeId) (GetBalancesRes, error) {
	if len(addrList) <= 0 || len(tokenIds) <= 0 {
		return nil, nil
	}

	res := make(map[types.Address]map[types.TokenTypeId]*big.Int)

	for _, tokenId := range tokenIds {
		//addrBalances[tokenId] = big.NewInt(1)
		balances, err := l.chain.GetConfirmedBalanceList(addrList, tokenId, snapshotHash)
		if err != nil {
			return nil, err
		}

		if balances == nil {
			return nil, errors.New(fmt.Sprintf("snapshot block %s is not existed.", snapshotHash))
		}

		for addr, balance := range balances {
			addrBalances, ok := res[addr]
			if !ok {
				addrBalances = make(map[types.TokenTypeId]*big.Int)
				res[addr] = addrBalances
			}
			addrBalances[tokenId] = balance
		}
	}

	return res, nil
}

func parseHeight(height interface{}) (uint64, error) {
	var heightUint64 uint64
	switch height.(type) {
	case float64:
		heightUint64 = uint64(height.(float64))

	case int:
		heightUint64 = uint64(height.(int))

	case uint64:
		heightUint64 = height.(uint64)

	case string:
		heightStr := height.(string)

		var err error
		heightUint64, err = strconv.ParseUint(heightStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return heightUint64, nil

}
