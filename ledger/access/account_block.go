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

func (acca *AccountChainAccess) GetAccountBlock (key []byte) (*ledger.AccountBlock, error) {
	//acca.store.Iterate()
	return nil, nil
}

func (acca *AccountChainAccess) GetAccountBalance (keyPartionList ...interface{}) (*big.Int, error) {
	// createkey
	key := vitedb.createKey(vitedb.DBKP_ACCOUNTBLOCK, keyPartionList,nil)
	accountBLock, err := acca.GetAccountBlock(key)
	if err != nil {
		return nil, nil
	}
	return accountBLock.Balance, nil
}