package util

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type dbInterface interface {
	GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error)
	SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
}

func AddBalance(db dbInterface, id *types.TokenTypeId, amount *big.Int) {
	b, err := db.GetBalance(id)
	DealWithErr(err)
	b.Add(b, amount)
	db.SetBalance(id, b)
}

func SubBalance(db dbInterface, id *types.TokenTypeId, amount *big.Int) {
	b, err := db.GetBalance(id)
	DealWithErr(err)
	if b.Cmp(amount) >= 0 {
		b.Sub(b, amount)
		db.SetBalance(id, b)
	}
}
