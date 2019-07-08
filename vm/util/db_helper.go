package util

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type dbInterface interface {
	GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error)
	SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int)
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) error
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

func GetValue(db dbInterface, key []byte) []byte {
	v, err := db.GetValue(key)
	DealWithErr(err)
	return v
}

func SetValue(db dbInterface, key []byte, value []byte) {
	err := db.SetValue(key, value)
	DealWithErr(err)
}
