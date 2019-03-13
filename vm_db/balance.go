package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (db *vmDB) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if balance, ok := db.unsaved.GetBalance(tokenTypeId); ok {
		return balance, nil
	}

	prevStateSnapshot, err := db.getPrevStateSnapshot()
	if err != nil {
		return nil, err
	}

	return prevStateSnapshot.GetBalance(tokenTypeId)
}
func (db *vmDB) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	db.unsaved.SetBalance(tokenTypeId, amount)
}
