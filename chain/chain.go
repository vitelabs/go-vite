package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/cache"
	"github.com/vitelabs/go-vite/chain/genesis"
	"github.com/vitelabs/go-vite/chain/index"
	"github.com/vitelabs/go-vite/chain/state"
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
 * 3. Init state database
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

			// Init state db
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
	//c.log.Error(cErr.Error(), "method", "checkAndRepair")

	// init cache
	if err := c.cache.Init(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Init failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "Init")
		return cErr
	}

	// init state db
	if err := c.stateDB.Init(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Init failed. Error: %s", err))
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

func (c *chain) checkAndRepair() error {
	// repair block db
	if err := c.blockDB.CheckAndRepair(); err != nil {
		return errors.New(fmt.Sprintf("c.blockDB.CheckAndRepair failed. Error: %s", err))
	}

	// repair index db
	indexDbLatestLocation, err := c.indexDB.QueryLatestLocation()
	if err != nil {
		return errors.New(fmt.Sprintf("c.indexDB.QueryLatestLocation failed. Error: %s", err))
	}
	if indexDbLatestLocation == nil {
		return errors.New(fmt.Sprintf("latestLocation is nil, Error: %s", err))
	}

	blockDbLatestLocation := c.blockDB.LatestLocation()
	compareResult := indexDbLatestLocation.Compare(blockDbLatestLocation)

	if compareResult < 0 {
		segs, err := c.blockDB.ReadRange(indexDbLatestLocation, blockDbLatestLocation)
		if err != nil {
			return errors.New(fmt.Sprintf("c.blockDB.ReadRange failed, startLocation is %+v, endLocation is %+v. Error: %s",
				indexDbLatestLocation, blockDbLatestLocation, err))
		}
		for _, seg := range segs {
			for _, block := range seg.AccountBlocks {
				if err := c.indexDB.InsertAccountBlock(block); err != nil {
					return errors.New(fmt.Sprintf("c.indexDB.InsertAccountBlock failed, block is %+v. Error: %s",
						block, err))
				}
			}
			if seg.SnapshotBlock != nil {
				if err := c.indexDB.InsertSnapshotBlock(
					seg.SnapshotBlock,
					seg.AccountBlocks,
					seg.SnapshotBlockLocation,
					seg.AccountBlocksLocation,
					nil,
					seg.RightBoundary,
				); err != nil {
					return errors.New(fmt.Sprintf("c.indexDB.InsertSnapshotBlock failed. Error: %s",
						err))
				}
			}
		}
	} else if compareResult > 0 {
		if err := c.indexDB.Rollback(blockDbLatestLocation); err != nil {
			return errors.New(fmt.Sprintf("c.indexDB.Rollback failed, location is %+v. Error: %s",
				blockDbLatestLocation, err))
		}
		//c.indexDB.DeleteSnapshotBlocks()
	}

	// repair state db
	//stateDbLatestLocation, err := c.stateDB.QueryLatestLocation()
	//
	//if err != nil {
	//	return errors.New(fmt.Sprintf("c.stateDB.QueryLatestLocation failed. Error: %s", err))
	//}
	//
	//blockDbLatestLocation := c.blockDB.LatestLocation()
	//
	//compareResult := stateDbLatestLocation.Compare(blockDbLatestLocation)
	//
	//if compareResult > 0 {
	//
	//} else if compareResult < 0 {
	//
	//}

	return nil
}
