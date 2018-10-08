package chain

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
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
		genesisSnapshotBlock: &GenesisSnapshotBlock,
		dataDir:              cfg.DataDir,
	}

	return chain
}

func (c *chain) Init() {
	c.log.Info("Init chain module")
	// stateTriePool
	c.stateTriePool = NewStateTriePool(c)

	// eventManager
	c.em = newEventManager()

	// chainDb
	chainDb := chain_db.NewChainDb(filepath.Join(c.dataDir, "ledger"))
	if chainDb == nil {
		c.log.Crit("NewChain failed, db init failed", "method", "Init")
	}
	c.chainDb = chainDb

	// compressor
	compressor := compress.NewCompressor(c, c.dataDir)
	c.compressor = compressor

	// check
	c.checkAndInitData()

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

	for addr := range unconfirmedSubLedger {
		helper.ReverseSlice(unconfirmedSubLedger[addr])
	}
	c.needSnapshotCache = NewNeedSnapshotContent(c, unconfirmedSubLedger)

	c.log.Info("Chain module initialized")
}
func (c *chain) checkAndInitData() {
	sb := c.genesisSnapshotBlock
	dbSb, err := c.GetSnapshotBlockByHeight(1)

	if err != nil || dbSb == nil || sb.Hash != dbSb.Hash {
		if err != nil {
			c.log.Error("GetSnapshotBlockByHeight failed, error is "+err.Error(), "method", "CheckAndInitDb")
		}

		c.clearData()
		c.initData()
		return
	}
}

func (c *chain) clearData() {
	// compressor clear
	err1 := c.compressor.ClearData()
	if err1 != nil {
		c.log.Crit("Compressor clear data failed, error is " + err1.Error())
	}
	// db clear
	err2 := c.chainDb.ClearData()
	if err2 != nil {
		c.log.Crit("ChainDb clear data failed, error is " + err2.Error())
	}
}

func (c *chain) initData() {
	// Write genesis snapshot block
	var err error

	// Insert mintage block
	err = c.InsertAccountBlocks([]*vm_context.VmAccountBlock{{
		AccountBlock: &GenesisMintageBlock,
		VmContext:    GenesisMintageBlockVC,
	}, {
		AccountBlock: &GenesisMintageSendBlock,
		VmContext:    GenesisMintageSendBlockVC,
	}})
	if err != nil {
		c.log.Crit("InsertGenesisMintageBlock failed, error is "+err.Error(), "method", "initData")
	}

	// Insert consensus group block
	err = c.InsertAccountBlocks([]*vm_context.VmAccountBlock{{
		AccountBlock: &GenesisConsensusGroupBlock,
		VmContext:    GenesisConsensusGroupBlockVC,
	}})
	if err != nil {
		c.log.Crit("InsertGenesisConsensusGroupBlock failed, error is "+err.Error(), "method", "initData")
	}

	// Insert register block
	err = c.InsertAccountBlocks([]*vm_context.VmAccountBlock{{
		AccountBlock: &GenesisRegisterBlock,
		VmContext:    GenesisRegisterBlockVC,
	}})
	if err != nil {
		c.log.Crit("InsertGenesisRegisterBlock failed, error is "+err.Error(), "method", "initData")
	}

	// Insert snapshot block
	err = c.InsertSnapshotBlock(c.genesisSnapshotBlock)
	if err != nil {
		c.log.Crit("WriteSnapshotBlock failed, error is "+err.Error(), "method", "initData")
	}

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
