package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (db *vmDB) GetBalance(tokenTypeId *types.TokenTypeId) *big.Int {
	return nil
}
func (db *vmDB) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {}
func (db *vmDB) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {}
