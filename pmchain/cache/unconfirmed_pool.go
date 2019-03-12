package chain_cache

import "github.com/vitelabs/go-vite/ledger"

type UnconfirmedPool struct {
}

func NewUnconfirmedPool() *UnconfirmedPool {
	return &UnconfirmedPool{}
}

func (up *UnconfirmedPool) InsertAccountBlock(block *ledger.AccountBlock) error {
	return nil
}
