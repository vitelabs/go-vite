package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (vdb *vmDb) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	if vdb.uns != nil {
		if balance, ok := vdb.unsaved().GetBalance(tokenTypeId); ok {
			return new(big.Int).Set(balance), nil
		}
	}

	return vdb.chain.GetBalance(*vdb.address, *tokenTypeId)
}

func (vdb *vmDb) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	vdb.unsaved().SetBalance(tokenTypeId, amount)
}

func (vdb *vmDb) GetUnsavedBalanceMap() map[types.TokenTypeId]*big.Int {
	if vdb.uns == nil {
		return make(map[types.TokenTypeId]*big.Int)
	}
	return vdb.unsaved().GetBalanceMap()
}
