package handler

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger/handler_interface"
)

func (ac *AccountChain) GetUnconfirmedTxHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error) {
	return ac.uAccess.GetUnconfirmedHashs(index, num, count, addr)
}

func (ac *AccountChain) GetUnconfirmedTxHashsByTkId(index, num, count int, addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	return ac.uAccess.GetUnconfirmedHashsByTkId(index, num, count, addr, tokenId)
}

func (ac *AccountChain) GetUnconfirmedAccount(addr *types.Address) (*handler_interface.UnconfirmedAccount, error) {
	unconfirmedMeta, err := ac.uAccess.GetUnconfirmedAccountMeta(addr)
	if err != nil {
		return nil, err
	}
	var tokenInfoList []*handler_interface.TokenInfo
	for _, ti := range unconfirmedMeta.TokenInfoList {
		token, tkErr := ac.tAccess.GetByTokenId(ti.TokenId)
		if tkErr != nil {
			return nil, tkErr
		}
		tokenInfo := &handler_interface.TokenInfo{
			Token:       token,
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
