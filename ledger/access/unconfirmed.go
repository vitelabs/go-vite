package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
)

type UnconfirmedAccess struct {
	store *vitedb.Unconfirmed
	//bwMutex sync.RWMutex
}

var unconfirmedAccess = &UnconfirmedAccess {
	store: vitedb.GetUnconfirmed(),
}

func GetUnconfirmedAccess () *UnconfirmedAccess {
	return unconfirmedAccess
}

func (ufb *UnconfirmedAccess) GetUnconfirmedBlocks (index int, num int, count int, addr *types.Address) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func (ufb *UnconfirmedAccess) GetUnconfirmedAccountMeta (addr *types.Address) (*ledger.UnconfirmedMeta, error) {
	return ufb.store.GetUnconfirmedMeta(addr)
}

func (ufb *UnconfirmedAccess) WriteBlock (addr *types.Address, hash *types.Hash) error {
	return nil
}

func (ufb *UnconfirmedAccess) DeleteBlock(addr *types.Address) error {

	return nil
}