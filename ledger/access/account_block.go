package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type AccountChainAccess struct {
	store *vitedb.AccountChain
}

func (AccountChainAccess) New () *AccountChainAccess {
	return &AccountChainAccess{
		store: vitedb.AccountChain{}.GetInstance(),
	}
}

// add by sanjin
func (AccountChain)GetAccountBalanceByKeyValue (accountBlockKey []byte) (*big.Int, error) {
	balance := big.NewInt(0)
	return balance, nil
}