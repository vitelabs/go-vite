package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
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
	//signer        *signer.Master
	//MintageCache map[types.TokenTypeId]*Mintage
	log log15.Logger
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

//
//func (l *LedgerApi) GetUnconfirmedBlocksByAccAddr(addr types.Address, index int, count int) ([]AccountBlock, error) {
//	log.Info("GetUnconfirmedBlocksByAccAddr")
//	blocks, e := l.ledgerManager.Ac().GetUnconfirmedTxBlocks(index, 1, count, &addr)
//	if e != nil {
//		return nil, e
//	}
//	if len(blocks) == 0 {
//		return nil, nil
//	}
//	result := make([]AccountBlock, len(blocks))
//	for key, value := range blocks {
//		result[key] = *LedgerAccBlockToRpc(value, nil)
//	}
//	return result, nil
//}
//
//func (l *LedgerApi) GetAccountByAccAddr(addr types.Address) (GetAccountResponse, error) {
//	log.Info("GetAccountByAccAddr")
//
//	account, err := l.ledgerManager.Ac().GetAccount(&addr)
//	if err != nil {
//		return GetAccountResponse{}, err
//	}
//
//	response := GetAccountResponse{}
//	if account == nil {
//		log.Error("account == nil")
//		return response, nil
//	}
//
//	if account.Address != nil {
//		response.Addr = *account.Address
//	}
//	if account.BlockHeight != nil {
//		response.BlockHeight = account.BlockHeight.String()
//	}
//
//	if len(account.TokenInfoList) != 0 {
//		var bs []BalanceInfo
//		bs = make([]BalanceInfo, len(account.TokenInfoList))
//		for i, v := range account.TokenInfoList {
//			amount := "0"
//			if v.TotalAmount != nil {
//				amount = v.TotalAmount.String()
//			}
//			bs[i] = BalanceInfo{
//				Mintage: rawMintageToRpc(v.Token),
//				Balance: amount,
//			}
//		}
//
//		response.BalanceInfos = bs
//	}
//	return response, nil
//}
//
//func (l *LedgerApi) GetUnconfirmedInfo(addr types.Address) (GetUnconfirmedInfoResponse, error) {
//	log.Info("GetUnconfirmedInfo")
//
//	account, e := l.ledgerManager.Ac().GetUnconfirmedAccount(&addr)
//	if e != nil {
//		log.Error(e.Error())
//		return GetUnconfirmedInfoResponse{}, e
//	}
//
//	response := GetUnconfirmedInfoResponse{}
//
//	if account == nil {
//		log.Error("account == nil")
//		return response, nil
//	}
//
//	if account.Address != nil {
//		response.Addr = *account.Address
//	}
//	if account.TotalNumber != nil {
//		response.UnConfirmedBlocksLen = account.TotalNumber.String()
//	}
//
//	if len(account.TokenInfoList) != 0 {
//		blances := make([]BalanceInfo, len(account.TokenInfoList))
//		for k, v := range account.TokenInfoList {
//			blances[k] = BalanceInfo{
//				Mintage: rawMintageToRpc(v.Token),
//				Balance: v.TotalAmount.String(),
//			}
//		}
//		response.BalanceInfos = blances
//
//	}
//
//	return response, nil
//
//}
//
//func (l *LedgerApi) GetInitSyncInfo() (InitSyncResponse, error) {
//	log.Info("GetInitSyncInfo")
//	i := l.ledgerManager.Sc().GetFirstSyncInfo()
//
//	r := InitSyncResponse{
//		StartHeight:      i.BeginHeight.String(),
//		TargetHeight:     i.TargetHeight.String(),
//		CurrentHeight:    i.CurrentHeight.String(),
//		IsFirstSyncDone:  i.IsFirstSyncDone,
//		IsStartFirstSync: i.IsFirstSyncStart,
//	}
//
//	return r, nil
//}
//
//func (l *LedgerApi) GetSnapshotChainHeight() (string, error) {
//	log.Info("GetSnapshotChainHeight")
//	block, e := l.ledgerManager.Sc().GetLatestBlock()
//	if e != nil {
//		log.Error(e.Error())
//		return "", e
//	}
//	if block != nil && block.Height != nil {
//		return block.Height.String(), nil
//	}
//	return "", nil
//}
//
//func (l *LedgerApi) GetLatestSnapshotChainHash() (*types.Hash, error) {
//	log.Info("GetLatestSnapshotChainHash")
//	block, e := l.ledgerManager.Sc().GetLatestBlock()
//	if e != nil {
//		log.Error(e.Error())
//		return nil, e
//	}
//	if block != nil && block.Hash != nil {
//		return block.Hash, nil
//	}
//	return nil, nil
//}
//
//func (l *LedgerApi) GetLatestBlock(addr types.Address, needToken *bool) (*AccountBlock, error) {
//	log.Info("GetLatestBlock")
//	b, getError := l.ledgerManager.Ac().GetLatestBlock(&addr)
//	if getError != nil {
//		return nil, getError.Err
//	}
//	return LedgerAccBlockToRpc(b, nil), nil
//}
//
//func (l *LedgerApi) SendTx(block *AccountBlock) error {
//	log.Info("SendTx")
//	if block == nil {
//		return errors.New("block nil")
//	}
//	accountBlock, e := block.ToLedgerAccBlock()
//	if e != nil {
//		return e
//	}
//	return l.ledgerManager.Ac().CreateTx(accountBlock)
//}
//
//func (l *LedgerApi) GetTokenMintage(tti types.TokenTypeId) (*Mintage, error) {
//	log.Info("GetTokenMintage")
//	token, e := l.ledgerManager.Ac().GetToken(tti)
//	if e != nil {
//		return nil, e
//	}
//	if token.Mintage == nil {
//		return nil, errors.New("token.Mintage nil")
//	}
//	return rawMintageToRpc(token.Mintage), nil
//
//}
//
////func (l *LedgerApi) StartAutoConfirmTx(addr []string, reply *string) error {
////	return nil
////}
////
////func (l *LedgerApi) StopAutoConfirmTx(addr []string, reply *string) error {
////	return nil
////}
//
//type PublicTxApi struct {
//	txApi *LedgerApi
//}
