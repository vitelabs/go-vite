package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"fmt"
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

func (aa *AccountAccess) GetAccountMeta (accountAddress *types.Address) (*ledger.AccountMeta, error){
	//return &ledger.AccountMeta {
	//	AccountId: big.NewInt(1),
	//	TokenList: []*ledger.AccountSimpleToken{{
	//		TokenId: []byte{1, 2, 3},
	//		LastAccountBlockHeight: big.NewInt(1),
	//	}},
	//}
	data, err := aa.store.GetAccountMetaByAddress(accountAddress)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return data, nil

}

