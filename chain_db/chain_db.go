package chain_db

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	errors2 "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/vitelabs/go-vite/chain_db/access"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/log15"
	"os"
)

type ChainDb struct {
	dbDir string
	db    *leveldb.DB

	Ac      *access.AccountChain
	Sc      *access.SnapshotChain
	Account *access.Account
	Be      *access.BlockEvent
	OnRoad  *access.OnRoad

	log log15.Logger
}

func NewChainDb(dbDir string) *ChainDb {
	cDb := &ChainDb{
		log: log15.New("module", "chainDb"),

		dbDir: dbDir,
	}

	err := cDb.initDb()
	if err != nil {
		cDb.log.Error("initDb failed, error is "+err.Error(), "method", "NewChainDb")
		return nil
	}

	return cDb
}

func (chainDb *ChainDb) initDb() error {
	db, err := database.NewLevelDb(chainDb.dbDir)
	if err != nil {
		switch err.(type) {
		case *errors2.ErrCorrupted:
			return chainDb.ClearData()
		default:
			chainDb.log.Error("NewLevelDb failed, error is "+err.Error(), "method", "initDb")
			return err
		}
	}

	if db == nil {
		err := errors.New("NewChainDb failed")
		chainDb.log.Error(err.Error(), "method", "initDb")
		return err
	}
	chainDb.db = db
	chainDb.Ac = access.NewAccountChain(db)
	chainDb.Sc = access.NewSnapshotChain(db)
	chainDb.Account = access.NewAccount(db)
	chainDb.Be = access.NewBlockEvent(db)
	chainDb.OnRoad = access.NewOnRoad(db)

	return nil
}

func (chainDb *ChainDb) ClearData() error {
	if chainDb.db != nil {
		if closeErr := chainDb.db.Close(); closeErr != nil {
			return errors.New("Close db failed, error is " + closeErr.Error())
		}
	}

	if err := os.RemoveAll(chainDb.dbDir); err != nil && err != os.ErrNotExist {
		return errors.New("Remove " + chainDb.dbDir + " failed, error is " + err.Error())
	}

	chainDb.db = nil
	return chainDb.initDb()
}

func (chainDb *ChainDb) Db() *leveldb.DB {
	return chainDb.db
}

func (chainDb *ChainDb) Commit(batch *leveldb.Batch) error {
	return chainDb.db.Write(batch, nil)
}
