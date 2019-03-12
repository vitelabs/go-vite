package pmchain

import (
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/cache"
	"github.com/vitelabs/go-vite/pmchain/genesis"
	"github.com/vitelabs/go-vite/pmchain/index"
)

type chain struct {
	genesis *chain_genesis.Genesis

	cache *chain_cache.Cache

	indexDB *chain_index.IndexDB

	blockDB *chain_block.BlockDB
}

/*
 * Init chain config
 */
func (c *chain) NewChain() *chain {
	return &chain{}
}

/*
 * 1. Check and init ledger (check genesis block)
 * 2. Init index database
 * 3. Init block database
 * 4. Init cache
 */
func (c *chain) Init() error {
	// Init ledger
	indexDB := chain_index.NewIndexDB()
	blockDB := chain_block.NewBlockDB()

	c.genesis = chain_genesis.NewGenesis(indexDB, blockDB)

	// Init index database
	c.indexDB = chain_index.NewIndexDB()

	// Init block database
	c.blockDB = chain_block.NewBlockDB()

	// Init cache
	c.cache = chain_cache.NewCache()

	return nil
}

func (c *chain) Start() error {
	return nil
}
func (c *chain) Stop() error {
	return nil
}

func (c *chain) Destroy() error {
	return nil
}
