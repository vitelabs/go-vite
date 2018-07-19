package handler

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
)

type UnconfirmedHandler struct {
	// Handle block
	store *vitedb.Unconfirmed
}

func NewUnconfirmedHandle () (*UnconfirmedHandler) {
	return &UnconfirmedHandler{
		store: vitedb.GetUnconfirmed(),
	}
}

func (ufb *UnconfirmedHandler) GetUnconfirmedBlocks (index int, num int, count int, accountAddress *types.Address) ([]*ledger.UnconfirmedBlock, error) {
	return nil, nil
}

func (ufb *UnconfirmedHandler) GetUnconfirmedAccount (accountAddress *types.Address) (*ledger.UnconfirmedMeta, error) {
	return nil, nil
}


func (ufb *UnconfirmedHandler) WriteBlockList () error {
	return nil
}

func (ufb *UnconfirmedHandler) WriteBlock ()  error {
	return nil
}

func (ufb *UnconfirmedHandler) writeBlock () error {
	return nil
}

