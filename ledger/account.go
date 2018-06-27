package ledger

import (
	"math/big"
)

type AccountSimpleToken struct {
	TokenId []byte
	LastAccountBlockHeight *big.Int
}

type AccountMeta struct {
	AccountId *big.Int
	TokenList []*AccountSimpleToken
}

type Account struct {
	AccountMeta
}

func (account *Account) GetBlockHeight () *big.Int {
	return big.NewInt(456)
}