package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"math/big"
)

type AccountAccess struct {
	store *vitedb.Account
}

var accountAccess = &AccountAccess {
	store: vitedb.GetAccount(),
}

func GetAccountAccess () *AccountAccess {
	return accountAccess
}

func (aa *AccountAccess) CreateNewAccountMeta (batch *leveldb.Batch, accountAddress *types.Address) (*ledger.AccountMeta, error) {
	// If account doesn't exist and the block is a response block, we must create account
	lastAccountID, err := aa.store.GetLastAccountId()
	if err != nil {
		return  nil, err
	}


	if lastAccountID == nil {
		lastAccountID = big.NewInt(0)
	}

	newAccountId := &big.Int{}
	newAccountId.Add(lastAccountID, big.NewInt(1))

	// Create account meta which will be write to database later
	accountMeta := &ledger.AccountMeta {
		AccountId: newAccountId,
		TokenList: []*ledger.AccountSimpleToken{},
	}

	return accountMeta, nil

}

func (aa *AccountAccess) GetAccountMeta (accountAddress *types.Address) (*ledger.AccountMeta, error){
	data, err := aa.store.GetAccountMetaByAddress(accountAddress)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return data, nil

}

