package chain

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/cache"
	"github.com/vitelabs/go-vite/chain/flusher"
	"github.com/vitelabs/go-vite/chain/genesis"
	"github.com/vitelabs/go-vite/chain/index"
	"github.com/vitelabs/go-vite/chain/state"
	"github.com/vitelabs/go-vite/chain/sync_cache"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_db"
	"os"
	"path"
	"sync/atomic"
)

const (
	stop  = 0
	start = 1
)

type chain struct {
	genesisCfg *config.Genesis

	genesisSnapshotBlock    *ledger.SnapshotBlock
	genesisAccountBlocks    []*vm_db.VmAccountBlock
	genesisAccountBlockHash map[types.Hash]struct{}

	dataDir   string
	chainDir  string
	consensus Consensus

	log log15.Logger

	em *eventManager

	cache *chain_cache.Cache

	indexDB *chain_index.IndexDB

	blockDB *chain_block.BlockDB

	stateDB *chain_state.StateDB

	syncCache interfaces.SyncCache

	flusher *chain_flusher.Flusher

	status uint32
}

/*
 * Init chain config
 */
func NewChain(dir string, chainCfg *config.Chain, genesisCfg *config.Genesis) *chain {
	c := &chain{
		genesisCfg: genesisCfg,
		dataDir:    dir,
		chainDir:   path.Join(dir, "ledger"),
		log:        log15.New("module", "chain"),
		em:         newEventManager(),
	}

	c.genesisAccountBlocks = chain_genesis.NewGenesisAccountBlocks(genesisCfg)
	c.genesisSnapshotBlock = chain_genesis.NewGenesisSnapshotBlock(c.genesisAccountBlocks)
	c.genesisAccountBlockHash = chain_genesis.VmBlocksToHashMap(c.genesisAccountBlocks)

	return c
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
		// Init ledger db
		if c.indexDB, err = chain_index.NewIndexDB(c.chainDir); err != nil {
			c.log.Error(fmt.Sprintf("chain_index.NewIndexDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
			return err
		}

		// Init block db
		if c.blockDB, err = chain_block.NewBlockDB(c.chainDir); err != nil {
			c.log.Error(fmt.Sprintf("chain_block.NewBlockDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "Init")
			return err
		}

		// Init state db
		if c.stateDB, err = chain_state.NewStateDB(c, c.chainDir); err != nil {
			cErr := errors.New(fmt.Sprintf("chain_cache.NewStateDB failed, error is %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

		// init flusher
		stores := []chain_flusher.Storage{c.blockDB, c.stateDB.StorageRedo(), c.stateDB.Store(), c.indexDB.Store()}
		if c.flusher, err = chain_flusher.NewFlusher(stores, c.chainDir); err != nil {
			cErr := errors.New(fmt.Sprintf("chain_flusher.NewFlusher failed. Error: %s", err))
			c.log.Error(cErr.Error(), "method", "Init")
			return cErr
		}

		// flusher check and recover
		if err := c.flusher.Recover(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.flusher.Recover failed. Error: %s", err))
			c.log.Error(cErr.Error(), "method", "Init")
			return cErr
		}

		// check ledger
		status, err := chain_genesis.CheckLedger(c, c.genesisSnapshotBlock)
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

			if status == chain_genesis.LedgerEmpty {
				// Init Ledger
				if err = chain_genesis.InitLedger(c, c.genesisSnapshotBlock, c.genesisAccountBlocks); err != nil {
					cErr := errors.New(fmt.Sprintf("chain_genesis.InitLedger failed, error is %s", err))
					c.log.Error(cErr.Error(), "method", "Init")
					return err
				}
			}
			break
		}
		// close blockDB
		if err = c.blockDB.Close(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.blockDB.Close failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

		// close indexDB
		if err = c.indexDB.Close(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.indexDB.Close failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

		// close stateDB
		if err = c.stateDB.Close(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.stateDB.Close failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

		// close flusher
		if err = c.flusher.Close(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.flusher.Close failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

		// clean all data
		if err = c.cleanAllData(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.cleanAllData failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "Init")
			return err
		}

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
	if !atomic.CompareAndSwapUint32(&c.status, stop, start) {
		return nil
	}

	return nil
}

func (c *chain) Stop() error {
	if !atomic.CompareAndSwapUint32(&c.status, start, stop) {
		return nil
	}

	return nil
}

func (c *chain) Destroy() error {
	c.log.Info("Begin to destroy", "method", "Close")

	c.cache.Destroy()
	c.log.Info("Close cache", "method", "Close")

	if err := c.stateDB.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Close failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Close")
		return cErr
	}
	c.log.Info("Close stateDB", "method", "Close")

	if err := c.indexDB.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Close failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Close")
		return cErr
	}
	c.log.Info("Close indexDB", "method", "Close")

	if err := c.blockDB.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Close failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Close")
		return cErr
	}
	c.log.Info("Close blockDB", "method", "Close")

	c.flusher = nil
	c.cache = nil
	c.stateDB = nil
	c.indexDB = nil
	c.blockDB = nil

	c.log.Info("Complete destruction", "method", "Close")

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

func (c *chain) SetConsensus(cs Consensus) {
	c.consensus = cs
}

func (c *chain) cleanAllData() error {
	return os.RemoveAll(c.chainDir)
}
