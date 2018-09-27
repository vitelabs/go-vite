package chain

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/trie"
	"path/filepath"
	"sync"
)

type chain struct {
	log        log15.Logger
	chainDb    *chain_db.ChainDb
	compressor *compress.Compressor

	trieNodePool  *trie.TrieNodePool
	stateTriePool *StateTriePool

	createAccountLock sync.Mutex

	needSnapshotCache *NeedSnapshotCache

	genesisSnapshotBlock *ledger.SnapshotBlock
	latestSnapshotBlock  *ledger.SnapshotBlock

	dataDir string

	em *eventManager
}

func NewChain(cfg *config.Config) Chain {
	chain := &chain{
		log:                  log15.New("module", "chain"),
		genesisSnapshotBlock: ledger.GetGenesisSnapshotBlock(),
		dataDir:              cfg.DataDir,
	}

	return chain
}

func (c *chain) Init() {
	c.log.Info("Init chain module")
	// stateTriePool
	c.stateTriePool = NewStateTriePool(c)

	// chainDb
	chainDb := chain_db.NewChainDb(filepath.Join(c.dataDir, "chain"))
	if chainDb == nil {
		c.log.Crit("NewChain failed, db init failed")
	}
	c.chainDb = chainDb

	// compressor
	compressor := compress.NewCompressor(c, c.dataDir)
	c.compressor = compressor

	// trieNodePool
	c.trieNodePool = trie.NewTrieNodePool()

	// latestSnapshotBlock
	var getLatestBlockErr error
	c.latestSnapshotBlock, getLatestBlockErr = c.chainDb.Sc.GetLatestBlock()
	if getLatestBlockErr != nil {
		c.log.Crit("GetLatestBlock failed, error is "+getLatestBlockErr.Error(), "method", "NewChain")
	}

	// needSnapshotCache
	unconfirmedSubLedger, getSubLedgerErr := c.getUnConfirmedSubLedger()
	if getSubLedgerErr != nil {
		c.log.Crit("getUnConfirmedSubLedger failed, error is "+getSubLedgerErr.Error(), "method", "NewChain")
	}

	c.needSnapshotCache = NewNeedSnapshotContent(c, unconfirmedSubLedger)

	// eventManager
	c.em = newEventManager()

	c.log.Info("Chain module initialized")
}

func (c *chain) Compressor() *compress.Compressor {
	return c.compressor
}

func (c *chain) ChainDb() *chain_db.ChainDb {
	return c.chainDb
}

func (c *chain) Start() {
	// Start compress in the background
	c.log.Info("Start chain module")

	c.compressor.Start()

	c.log.Info("Chain module started")
}

func (c *chain) Stop() {
	// Stop compress
	c.log.Info("Stop chain module")

	c.compressor.Stop()

	c.log.Info("Chain module stopped")
}

func (c *chain) destroy() {
	c.log.Info("Destroy chain module")
	// stateTriePool
	c.stateTriePool = nil

	// close db
	c.chainDb.Db().Close()
	c.chainDb = nil

	// compressor
	c.compressor = nil

	// trieNodePool
	c.trieNodePool = nil

	// needSnapshotCache
	c.needSnapshotCache = nil

	c.log.Info("Chain module destroyed")
}
