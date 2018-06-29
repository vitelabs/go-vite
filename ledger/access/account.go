package access

import (
	"github.com/vitelabs/go-vite/vitedb"
)

type AccountAccess struct {
	store *vitedb.Account
}

func (AccountAccess) New () *AccountAccess {
	return &AccountAccess{
		store: vitedb.Account{}.GetInstance(),
	}
}