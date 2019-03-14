package pmchain

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/cache"
	"github.com/vitelabs/go-vite/pmchain/genesis"
	"github.com/vitelabs/go-vite/pmchain/index"
	"github.com/vitelabs/go-vite/pmchain/state"
)

type chain struct {
	log   log15.Logger
	cache *chain_cache.Cache

	indexDB *chain_index.IndexDB

	blockDB *chain_block.BlockDB

	stateDB *chain_state.StateDB
}

/*
 * Init chain config
 */
func (c *chain) NewChain() Chain {
	return &chain{
		log: log15.New("module", "chain"),
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
	// Init ledger
	chainDir := ""
	indexDB, err := chain_index.NewIndexDB(chainDir)
	if err != nil {
		c.log.Error(fmt.Sprintf("Init failed, call chain_index.NewIndexDB error, error is %s, chainDir is %s", err, chainDir))
		return err
	}
	//stateDB := chain_state.NewStateDB()
	blockDB := chain_block.NewBlockDB()

	if !chain_genesis.CheckLedger(indexDB, blockDB) {
		// destroy
		indexDB.Destroy()
		blockDB.Destroy()

		// init
		indexDB, err = chain_index.NewIndexDB(chainDir)
		if err != nil {
			c.log.Error(fmt.Sprintf("Init failed, call chain_index.NewIndexDB error, error is %s, chainDir is %s", err, chainDir))
			return err
		}

		blockDB = chain_block.NewBlockDB()

		c.log.Info("Init ledger", "method", "Init")
		chain_genesis.InitLedger(indexDB, blockDB)
	}

	// Init index database
	c.indexDB = indexDB

	// Init block database
	c.blockDB = blockDB

	// Init cache
	c.cache = chain_cache.NewCache()

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

	c.indexDB.Destroy()
	c.log.Info("Destroy indexDB", "method", "Destroy")

	c.blockDB.Destroy()
	c.log.Info("Destroy blockDB", "method", "Destroy")

	c.cache = nil
	c.indexDB = nil
	c.blockDB = nil

	c.log.Info("Complete destruction", "method", "Destroy")

	return nil
}
