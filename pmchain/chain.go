package pmchain

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/cache"
	"github.com/vitelabs/go-vite/pmchain/genesis"
	"github.com/vitelabs/go-vite/pmchain/index"
	"github.com/vitelabs/go-vite/pmchain/state"
	"path"
)

type chain struct {
	dataDir  string
	chainDir string

	log log15.Logger

	cache *chain_cache.Cache

	indexDB *chain_index.IndexDB

	blockDB *chain_block.BlockDB

	stateDB *chain_state.StateDB
}

/*
 * Init chain config
 */
func NewChain(dataDir string) Chain {
	return &chain{
		dataDir:  dataDir,
		chainDir: path.Join(dataDir, "ledger"),
		log:      log15.New("module", "chain"),
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
	indexDB, err := chain_index.NewIndexDB(c.chainDir)
	if err != nil {
		c.log.Error(fmt.Sprintf("chain_index.NewIndexDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
		return err
	}
	//stateDB := chain_state.NewStateDB()
	blockDB, err := chain_block.NewBlockDB(c.chainDir)
	if err != nil {
		c.log.Error(fmt.Sprintf("chain_block.NewBlockDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
		return err
	}

	if !chain_genesis.CheckLedger(indexDB, blockDB) {
		// destroy
		indexDB.Destroy()
		blockDB.Destroy()

		// init
		indexDB, err = chain_index.NewIndexDB(c.chainDir)
		if err != nil {
			c.log.Error(fmt.Sprintf("chain_index.NewIndexDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
			return err
		}

		blockDB, err = chain_block.NewBlockDB(c.dataDir)
		if err != nil {
			c.log.Error(fmt.Sprintf("chain_block.NewBlockDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
			return err
		}

		c.log.Info("Init ledger", "method", "Init")
		chain_genesis.InitLedger(indexDB, blockDB)
	}

	// Init index database
	c.indexDB = indexDB

	// Init block database
	c.blockDB = blockDB

	// Init cache
	if c.cache, err = chain_cache.NewCache(c); err != nil {
		c.log.Error(fmt.Sprintf("chain_cache.NewCache failed, error is %s", err), "method", "Init")
		return err
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
