package ledger

import (
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
)

type AccountSimpleToken struct {
	TokenId *types.TokenTypeId
	LastAccountBlockHeight *big.Int
}

type AccountMeta struct {
	AccountId *big.Int
	TokenList []*AccountSimpleToken
}

type Account struct {
	AccountMeta
	blockHeight *big.Int
}

func (account *Account) GetBlockHeight () *big.Int {
	return big.NewInt(456)
}

