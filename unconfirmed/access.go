package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitedb"
)

type Unconfirmed_interface interface {
	WriteUnconfirmedMeta(block *ledger.AccountBlock) error
	DeleteUnconfirmedMeta(block *ledger.AccountBlock) error
	GetUnconfirmedHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error)
}

type UnconfirmedAccess struct {
	store *vitedb.Unconfirmed
}

func (self *UnconfirmedAccess) WriteUnconfirmed(block *ledger.AccountBlock) error {
	return nil
}

func (self *UnconfirmedAccess) WriteUnconfirmedMeta(block *ledger.AccountBlock) error {
	return nil
}

func (self *UnconfirmedAccess) DeleteUnconfirmedMeta(block *ledger.AccountBlock) error {
	return nil
}

func (self *UnconfirmedAccess) GetUnconfirmedHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error) {
	return nil, nil
}
