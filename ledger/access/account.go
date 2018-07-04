package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"github.com/vitelabs/go-vite/common/types"
)

type AccountAccess struct {
	store *vitedb.Account
}

var _accountAccess *AccountAccess

func GetAccountAccess () *AccountAccess {
	if _accountAccess == nil {
		_accountAccess = &AccountAccess {
			store: vitedb.GetAccount(),
		}
	}

	return _accountAccess
}

func (aa *AccountAccess) GetAccountMeta (accountAddress string) (*ledger.AccountMeta, error){
	//return &ledger.AccountMeta {
	//	AccountId: big.NewInt(1),
	//	TokenList: []*ledger.AccountSimpleToken{{
	//		TokenId: []byte{1, 2, 3},
	//		LastAccountBlockHeight: big.NewInt(1),
	//	}},
	//}
	hexAddress, h2Err := types.HexToAddress(accountAddress)
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

