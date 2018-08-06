package handler

import (
	"github.com/inconshreveable/log15"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
	"math/big"
)

var adLog = log15.New("module", "ledger/handler/account_chain")

func (ac *AccountChain) GetAccount(accountAddress *types.Address) (*handler_interface.Account, error) {
	accountMeta, err := ac.aAccess.GetAccountMeta(accountAddress)
	if err != nil {
		adLog.Info("func GetAccount.GetAccountMeta failed, ", "err", err)
		return nil, nil
	}
	accountBLockHeight, err := ac.acAccess.GetLatestBlockHeightByAccountId(accountMeta.AccountId)
	if err != nil {
		adLog.Info("func GetAccount.GetLatestBlockHeightByAccountId failed,", "err", err)
		return nil, nil
	}
	accountTokenList, err := ac.GetAccountTokenList(accountMeta)
	if err != nil {
		adLog.Info("func GetAccount.GetAccountTokenList failed, ", "err", err)
		return nil, nil
	}
	return NewAccount(accountAddress, accountBLockHeight, accountTokenList), nil
}

func (ac *AccountChain) GetAccountTokenList(accountMeta *ledger.AccountMeta) ([]*handler_interface.TokenInfo, error) {
	accountId := accountMeta.AccountId
	var accountTokenList []*handler_interface.TokenInfo
	for _, accountSimpleToken := range accountMeta.TokenList {
		accountToken, err := ac.GetAccountToken(accountSimpleToken.TokenId, accountId,
			accountSimpleToken.LastAccountBlockHeight)
		if err != nil {
			return nil, err
		}
		accountTokenList = append(accountTokenList, accountToken)
	}
	return accountTokenList, nil
}

func (ac *AccountChain) GetAccountToken(tokenId *types.TokenTypeId, accountId *big.Int, blockHeight *big.Int) (*handler_interface.TokenInfo, error) {
	token, err := ac.tAccess.GetByTokenId(tokenId)
	if err != nil {
		return nil, err
	}
	balance, balanceErr := ac.acAccess.GetAccountBalance(accountId, blockHeight)
	if balanceErr != nil {
		return nil, err
	}
	return NewAccountToken(token.Mintage, balance), nil
}

func NewAccount(accountAddress *types.Address, blockHeight *big.Int, accountTokenList []*handler_interface.TokenInfo) *handler_interface.Account {
	return &handler_interface.Account{
		AccountAddress: accountAddress,
		BlockHeight:    blockHeight,
		TokenInfoList:  accountTokenList,
	}
}

func NewAccountToken(token *ledger.Mintage, balance *big.Int) *handler_interface.TokenInfo {
	return &handler_interface.TokenInfo{
		TotalAmount: balance,
		Token:       token,
	}
}

func (ac *AccountChain) GetUnconfirmedTxHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error) {
	return ac.uAccess.GetUnconfirmedHashs(index, num, count, addr)
}

func (ac *AccountChain) GetUnconfirmedTxHashsByTkId(index, num, count int, addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	return ac.uAccess.GetUnconfirmedHashsByTkId(index, num, count, addr, tokenId)
}

func (ac *AccountChain) GetUnconfirmedAccount(addr *types.Address) (*handler_interface.UnconfirmedAccount, error) {
	unconfirmedMeta, err := ac.uAccess.GetUnconfirmedAccountMeta(addr)
	if err != nil {
		adLog.Info("func GetUnconfirmedAccount.GetUnconfirmedAccountMeta failed, error: ", err)
		return nil, nil
	}
	var tokenInfoList []*handler_interface.TokenInfo
	for _, ti := range unconfirmedMeta.TokenInfoList {
		token, tkErr := ac.tAccess.GetByTokenId(ti.TokenId)
		if tkErr != nil {
			adLog.Info("func GetUnconfirmedAccount.GetByTokenId failed, error: ", tkErr)
			return nil, nil
		}
		tokenInfo := &handler_interface.TokenInfo{
			Token:       token.Mintage,
			TotalAmount: ti.TotalAmount,
		}

		tokenInfoList = append(tokenInfoList, tokenInfo)
	}
	var UnconfirmedAccount = &handler_interface.UnconfirmedAccount{
		AccountAddress: addr,
		TotalNumber:    unconfirmedMeta.TotalNumber,
		TokenInfoList:  tokenInfoList,
	}
	return UnconfirmedAccount, nil
}

func (ac *AccountChain) AddListener(addr types.Address, change chan<- struct{}) {
	ac.uAccess.AddListener(addr, change)
}

func (ac *AccountChain) RemoveListener(addr types.Address) {
	ac.uAccess.RemoveListener(addr)
}
