package vitedb

import (
	"math/big"
	"github.com/vitelabs/go-vite/ledger"
	"log"
)

type Account struct {
	db *DataBase
}

var _account *Account

func (account Account) GetInstance () *Account {
	if _account == nil {
		db, err:= GetLDBDataBase(DB_BLOCK)
		if err != nil {
			log.Fatal(err)
		}

		_account = &Account{
			db: db,
		}
	}

	return _account

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