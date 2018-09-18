package chain_db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/access"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/log15"
)

type ChainDb struct {
	db *leveldb.DB

	Ac      *access.AccountChain
	Sc      *access.SnapshotChain
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
		db:      db,
		Ac:      access.NewAccountChain(db),
		Sc:      access.NewSnapshotChain(db),
		Account: access.NewAccount(db),
	}
}

func (chainDb *ChainDb) Db() *leveldb.DB {
	return chainDb.db
}

func (chainDb *ChainDb) Commit(batch *leveldb.Batch) error {
	return chainDb.db.Write(batch, nil)
}
