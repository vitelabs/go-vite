package access

import (
	"github.com/vitelabs/go-vite/vitedb"
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


func (acca *AccountChainAccess) GetAccountBalance (keyPartionList ...interface{}) (*big.Int, error) {
	key, err := vitedb.createKey(vitedb.DBKP_ACCOUNTBLOCK, keyPartionList,nil)
	if err != nil {
		return nil, err
	}
	accountBLock, err := acca.store.GetAccountBlock(key)
	if err != nil {
		return nil, err
	}
	return accountBLock.Balance, nil
}