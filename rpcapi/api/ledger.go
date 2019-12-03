package api

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
)

func NewLedgerApi(vite *vite.Vite) *LedgerApi {
	api := &LedgerApi{
		vite:  vite,
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
	vite  *vite.Vite
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

// old api
func (l *LedgerApi) GetBlockByHash(blockHash types.Hash) (*AccountBlock, error) {
	return l.GetAccountBlockByHash(blockHash)
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

// old api
func (l *LedgerApi) GetBlocksByHash(addr types.Address, originBlockHash *types.Hash, count uint64) ([]*AccountBlock, error) {
	return l.GetAccountBlocks(addr, originBlockHash, nil, count)
}

// in token
func (l *LedgerApi) GetBlocksByHashInToken(addr types.Address, originBlockHash *types.Hash, tokenTypeId types.TokenTypeId, count uint64) ([]*AccountBlock, error) {
	return l.GetAccountBlocks(addr, originBlockHash, &tokenTypeId, count)
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

// old api
func (l *LedgerApi) GetBlockByHeight(addr types.Address, height interface{}) (*AccountBlock, error) {
	return l.GetAccountBlockByHeight(addr, height)
}

// old api
func (l *LedgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int) ([]*AccountBlock, error) {
	return l.GetAccountBlocksByAddress(addr, index, count)
}

// old api
func (l *LedgerApi) GetAccountByAccAddr(addr types.Address) (*RpcAccountInfo, error) {
	info, err := l.getAccountInfoByAddress(addr)
	if err != nil {
		return nil, err
	}
	return ToRpcAccountInfo(l.chain, info), nil
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

// old api
func (l *LedgerApi) GetLatestSnapshotChainHash() *types.Hash {
	return l.GetLatestSnapshotHash()
}

func (l *LedgerApi) GetLatestSnapshotBlock() (*SnapshotBlock, error) {
	block := l.chain.GetLatestSnapshotBlock()
	return l.ledgerSnapshotBlockToRpcBlock(block)
}

// old api
func (l *LedgerApi) GetLatestBlock(addr types.Address) (*AccountBlock, error) {
	return l.GetLatestAccountBlock(addr)
}

// old api
func (l *LedgerApi) GetVmLogList(blockHash types.Hash) (ledger.VmLogList, error) {
	return l.GetVmLogs(blockHash)
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
