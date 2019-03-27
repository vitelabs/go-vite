package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/cache"
	"github.com/vitelabs/go-vite/chain/genesis"
	"github.com/vitelabs/go-vite/chain/index"
	"github.com/vitelabs/go-vite/chain/state"
	"github.com/vitelabs/go-vite/chain/sync_cache"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/log15"
	"path"
)

type chain struct {
	dataDir  string
	chainDir string

	log log15.Logger

	em *eventManager

	cache *chain_cache.Cache

	indexDB *chain_index.IndexDB

	blockDB *chain_block.BlockDB

	stateDB *chain_state.StateDB

	syncCache interfaces.SyncCache
}

/*
 * Init chain config
 */
func NewChain(dataDir string) *chain {
	return &chain{
		dataDir:  dataDir,
		chainDir: path.Join(dataDir, "ledger"),
		log:      log15.New("module", "chain"),
		em:       newEventManager(),
	}
}

/*
 * 1. Check and init ledger (check genesis block)
 * 2. Init index database
 * 3. Init state_bak database
 * 4. Init block database
 * 5. Init cache
 */
func (c *chain) Init() error {
	c.log.Info("Begin initializing", "method", "Init")
	for {
		var err error
		// Init ledger
		c.indexDB, err = chain_index.NewIndexDB(c, c.chainDir)
		if err != nil {
			c.log.Error(fmt.Sprintf("chain_index.NewIndexDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
			return err
		}

		c.blockDB, err = chain_block.NewBlockDB(c.chainDir)
		if err != nil {
			c.log.Error(fmt.Sprintf("chain_block.NewBlockDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
			return err
		}

		status, err := chain_genesis.CheckLedger(c)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("chain_genesis.CheckLedger failed, error is %s, chainDir is %s", err, c.chainDir))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}
		if status != chain_genesis.LedgerInvalid {
			// valid or empty
			// Init cache
			if c.cache, err = chain_cache.NewCache(c); err != nil {
				cErr := errors.New(fmt.Sprintf("chain_cache.NewCache failed, error is %s", err))

				c.log.Error(cErr.Error(), "method", "Init")
				return err
			}

			// Init state_bak db
			if c.stateDB, err = chain_state.NewStateDB(c, c.chainDir); err != nil {
				cErr := errors.New(fmt.Sprintf("chain_cache.NewStateDB failed, error is %s", err))

				c.log.Error(cErr.Error(), "method", "Init")
				return err
			}

			if status == chain_genesis.LedgerEmpty {
				// Init Ledger
				if err = chain_genesis.InitLedger(c); err != nil {
					cErr := errors.New(fmt.Sprintf("chain_genesis.InitLedger failed, error is %s", err))
					c.log.Error(cErr.Error(), "method", "Init")
					return err
				}
			} else {
				// Check and repair
				if err = c.checkAndRepair(); err != nil {
					cErr := errors.New(fmt.Sprintf("c.checkAndRepair failed, error is %s", err))
					c.log.Error(cErr.Error(), "method", "Init")
					return err
				}
			}

			break

		}

		// clean
		if err = c.indexDB.CleanAllData(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.indexDB.CleanAllData failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}
		if err = c.blockDB.CleanAllData(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.blockDB.CleanAllData failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

		// destroy
		c.indexDB.Destroy()
		c.blockDB.Destroy()
	}

	// init cache
	if err := c.cache.Init(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Init failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "Init")
		return cErr
	}

	// init sync cache
	var err error
	c.syncCache, err = sync_cache.NewSyncCache(c.chainDir)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("sync_cache.NewSyncCache failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "Init")
		return cErr
	}

	c.log.Info("Complete initialization", "method", "Init")
	return nil
}

func (c *chain) Start() error {
	return nil
}
func (c *chain) Stop() error {
	return nil
}

func (c *chain) Destroy() error {
	c.log.Info("Begin to destroy", "method", "Destroy")

	c.cache.Destroy()
	c.log.Info("Destroy cache", "method", "Destroy")

	if err := c.stateDB.Destroy(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Destroy failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Destroy")
		return cErr
	}
	c.log.Info("Destroy stateDB", "method", "Destroy")

	if err := c.indexDB.Destroy(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Destroy failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Destroy")
		return cErr
	}
	c.log.Info("Destroy indexDB", "method", "Destroy")

	if err := c.blockDB.Destroy(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Destroy failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Destroy")
		return cErr
	}
	c.log.Info("Destroy blockDB", "method", "Destroy")

	c.cache = nil
	c.stateDB = nil
	c.indexDB = nil
	c.blockDB = nil

	c.log.Info("Complete destruction", "method", "Destroy")

	return nil
}

func (c *chain) NewDb(dirName string) (*leveldb.DB, error) {
	absoluteDirName := path.Join(c.chainDir, dirName)
	db, err := leveldb.OpenFile(absoluteDirName, nil)

	if err != nil {
		return nil, err
	}
	return db, nil
}
