package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (db *vmDb) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.unsaved.GetBalance(tokenTypeId); ok {
		return new(big.Int).Set(balance), nil
	}

	return db.chain.GetBalance(*db.address, *tokenTypeId)
}
func (db *vmDb) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	db.unsaved.SetBalance(tokenTypeId, amount)
}

func (db *vmDb) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	return db.unsaved.GetBalanceMap()
}
