package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"log"
)

type AccountAccess struct {
	store *vitedb.Account
}

func (AccountAccess) New () *AccountAccess {
	return &AccountAccess{
		store: vitedb.Account{}.GetInstance(),
	}
}

func (aa *AccountAccess) GetAccountMeta (accountAddress []byte) (*ledger.AccountMeta, error){
	//return &ledger.AccountMeta {
	//	AccountId: big.NewInt(1),
	//	TokenList: []*ledger.AccountSimpleToken{{
	//		TokenId: []byte{1, 2, 3},
	//		LastAccountBlockHeight: big.NewInt(1),
	//	}},
	//}
	data, err := aa.store.GetAccountMetaByAddress(accountAddress)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &data, nil
	return nil,nil
}