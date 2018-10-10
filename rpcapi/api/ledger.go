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

func (l *LedgerApi) ledgerBlocksToRpcBlocks(list []*ledger.AccountBlock) ([]*AccountBlock, error) {
	var blocks []*AccountBlock
	for _, item := range list {
		confirmTimes, err := l.chain.GetConfirmTimes(&item.Hash)

		if err != nil {
			return nil, err
		}

		token := l.chain.GetTokenInfoById(&item.TokenId)
		blocks = append(blocks, createAccountBlock(item, token, confirmTimes))
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

func (l *LedgerApi) GetUnconfirmedBlocksByAccAddr(addr types.Address, index int, count int) ([]AccountBlock, error) {
	//log.Info("GetUnconfirmedBlocksByAccAddr")
	//
	//blocks, e := l.ledgerManager.Ac().GetUnconfirmedTxBlocks(index, 1, count, &addr)
	//if e != nil {
	//	return nil, e
	//}
	//if len(blocks) == 0 {
	//	return nil, nil
	//}
	//
	//result := make([]AccountBlock, len(blocks))
	//for key, value := range blocks {
	//	result[key] = *LedgerAccBlockToRpc(value, nil)
	//}
	//return result, nil
	return nil, nil
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
			TokenInfo:   RawTokenInfoToRpc(token),
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

func (l *LedgerApi) GetUnconfirmedInfo(addr types.Address) error {
	//log.Info("GetUnconfirmedInfo")
	//
	//account, e := l.ledgerManager.Ac().GetUnconfirmedAccount(&addr)
	//if e != nil {
	//	log.Error(e.Error())
	//	return GetUnconfirmedInfoResponse{}, e
	//}
	//
	//response := GetUnconfirmedInfoResponse{}
	//
	//if account == nil {
	//	log.Error("account == nil")
	//	return response, nil
	//}
	//
	//if account.Address != nil {
	//	response.Addr = *account.Address
	//}
	//if account.TotalNumber != nil {
	//	response.UnConfirmedBlocksLen = account.TotalNumber.String()
	//}
	//
	//if len(account.TokenInfoList) != 0 {
	//	blances := make([]BalanceInfo, len(account.TokenInfoList))
	//	for k, v := range account.TokenInfoList {
	//		blances[k] = BalanceInfo{
	//			Mintage: rawMintageToRpc(v.Token),
	//			Balance: v.TotalAmount.String(),
	//		}
	//	}
	//	response.BalanceInfos = blances
	//
	//}

	return nil

}

func (l *LedgerApi) GetInitSyncInfo() error {
	log.Info("GetInitSyncInfo")
	//i := l.ledgerManager.Sc().GetFirstSyncInfo()
	//
	//r := InitSyncResponse{
	//	StartHeight:      i.BeginHeight.String(),
	//	TargetHeight:     i.TargetHeight.String(),
	//	CurrentHeight:    i.CurrentHeight.String(),
	//	IsFirstSyncDone:  i.IsFirstSyncDone,
	//	IsStartFirstSync: i.IsFirstSyncStart,
	//}

	return nil
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

	token := l.chain.GetTokenInfoById(&block.TokenId)
	rpcBlock := createAccountBlock(block, token, 0)
	return rpcBlock, nil
}

func (l *LedgerApi) GetTokenMintage(tti types.TokenTypeId) (*RpcTokenInfo, error) {
	l.log.Info("GetTokenMintage")
	return RawTokenInfoToRpc(l.chain.GetTokenInfoById(&tti)), nil

}
