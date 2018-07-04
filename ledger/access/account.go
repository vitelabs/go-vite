package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"github.com/vitelabs/go-vite/common"
)

type AccountAccess struct {
	store *vitedb.Account
}

func (AccountAccess) New () *AccountAccess {
	return &AccountAccess{
		store: vitedb.Account{}.GetInstance(),
	}
}

func (aa *AccountAccess) GetAccountMeta (accountAddress string) (*ledger.AccountMeta, error){
	//return &ledger.AccountMeta {
	//	AccountId: big.NewInt(1),
	//	TokenList: []*ledger.AccountSimpleToken{{
	//		TokenId: []byte{1, 2, 3},
	//		LastAccountBlockHeight: big.NewInt(1),
	//	}},
	//}
	hexAddress, h2Err := common.HexToAddress(accountAddress)
	if h2Err != nil {
		return nil, h2Err
	}
	data, err := aa.store.GetAccountMetaByAddress(&hexAddress)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return data, nil

}

