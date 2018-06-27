package vitedb

import (
	"math/big"
	"github.com/vitelabs/go-vite/ledger"
)

type Account struct {
	db *DataBase
}

func (account Account) New () *Account {
	db := GetDataBase(DB_BLOCK)
	return &Account{
		db: db,
	}
}


func (account *Account) GetAccountMeta (accountAddress []byte) *ledger.AccountMeta {
	return &ledger.AccountMeta {
		AccountId: big.NewInt(1),
		TokenList: []*ledger.AccountSimpleToken{{
			TokenId: []byte{1, 2, 3},
			LastAccountBlockHeight: big.NewInt(1),
		}},
	}
}