package ledger

import "math/big"

type AccountMeta struct {
	AccountId *big.Int
	TokenList []string
}
