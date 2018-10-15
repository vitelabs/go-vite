package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"strconv"
)

// !!! Block = Transaction = TX

func NewLedgerApi(vite *vite.Vite) *LedgerApi {
	return &LedgerApi{
		chain: vite.Chain(),
		//signer:        vite.Signer(),
		log: log15.New("module", "rpc_api/ledger_api"),
	}
}

type LedgerApi struct {
	chain chain.Chain
	log   log15.Logger
}

func (l LedgerApi) String() string {
	return "LedgerApi"
}

func (l *LedgerApi) ledgerBlockToRpcBlock(block *ledger.AccountBlock) (*AccountBlock, error) {
	confirmTimes, err := l.chain.GetConfirmTimes(&block.Hash)

	if err != nil {
		return nil, err
	}

	var fromAddress, toAddress types.Address
	if block.IsReceiveBlock() {
		toAddress = block.AccountAddress
		sendBlock, err := l.chain.GetAccountBlockByHash(&block.FromBlockHash)
		if err != nil {
			return nil, err
		}

		if sendBlock != nil {
			fromAddress = sendBlock.AccountAddress
			block.TokenId = sendBlock.TokenId
			block.Amount = sendBlock.Amount
		}
	} else {
		fromAddress = block.AccountAddress
		toAddress = block.ToAddress
	}

	token := l.chain.GetTokenInfoById(&block.TokenId)
	rpcAccountBlock := createAccountBlock(block, token, confirmTimes)
	rpcAccountBlock.FromAddress = fromAddress
	rpcAccountBlock.ToAddress = toAddress
	return rpcAccountBlock, nil
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

func (l *LedgerApi) GetBlocksByHash(addr types.Address, originBlockHash *types.Hash, count uint64) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByHash")

	list, getError := l.chain.GetAccountBlocksByHash(addr, originBlockHash, count, false)
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

func (l *LedgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int, needTokenInfo *bool) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByAccAddr")

	list, getErr := l.chain.GetAccountBlocksByAddress(&addr, index, 1, count)

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

	account, err := l.chain.GetAccount(&addr)
	if err != nil {
		l.log.Error("GetAccount failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	if account == nil {
		return nil, nil
	}

	latestAccountBlock, err := l.chain.GetLatestAccountBlock(&addr)
	if err != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	totalNum := uint64(0)
	if latestAccountBlock != nil {
		totalNum = latestAccountBlock.Height
	}

	balanceMap, err := l.chain.GetAccountBalance(&addr)
	if err != nil {
		l.log.Error("GetAccountBalance failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	tokenBalanceInfoMap := make(map[types.TokenTypeId]*RpcTokenBalanceInfo)
	for tokenId, amount := range balanceMap {
		token := l.chain.GetTokenInfoById(&tokenId)
		tokenBalanceInfoMap[tokenId] = &RpcTokenBalanceInfo{
			TokenInfo:   RawTokenInfoToRpc(token, tokenId),
			TotalAmount: amount.String(),
			Number:      nil,
		}
	}

	rpcAccount := &RpcAccountInfo{
		AccountAddress:      account.AccountAddress,
		TotalNumber:         strconv.FormatUint(totalNum, 10),
		TokenBalanceInfoMap: tokenBalanceInfoMap,
	}

	return rpcAccount, nil
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
	block, getError := l.chain.GetLatestAccountBlock(&addr)
	if getError != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+getError.Error(), "method", "GetLatestBlock")
		return nil, getError
	}

	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

func (l *LedgerApi) GetTokenMintage(tti types.TokenTypeId) (*RpcTokenInfo, error) {
	l.log.Info("GetTokenMintage")
	return RawTokenInfoToRpc(l.chain.GetTokenInfoById(&tti), tti), nil
}
