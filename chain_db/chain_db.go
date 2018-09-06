package chain_db

import (
	"github.com/vitelabs/go-vite/chain_db/access"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/log15"
)

type ChainDb struct {
	Ac      *access.AccountChain
	Account *access.Account
}

var chainDbLog = log15.New("module", "chainDb")

func NewChainDb(dbDir string) *ChainDb {
	db := database.NewLevelDb(dbDir)
	if db == nil {
		chainDbLog.Error("NewChainDb failed")
		return nil
	}

	return &ChainDb{
		Ac:      access.NewAccountChain(db),
		Account: access.NewAccount(db),
	}
}
