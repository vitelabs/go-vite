package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/cache"
	"github.com/vitelabs/go-vite/chain/index"
	"github.com/vitelabs/go-vite/chain/sender"
	"github.com/vitelabs/go-vite/chain/trie_gc"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
	"path/filepath"
	"reflect"
	"sync"
)

type chain struct {
	log        log15.Logger
	blackBlock *blackBlock

	chainDb    *chain_db.ChainDb
	compressor *compress.Compressor

	trieNodePool  *trie.TrieNodePool
	stateTriePool *StateTriePool

	createAccountLock sync.Mutex

	needSnapshotCache *chain_cache.NeedSnapshotCache

	genesisSnapshotBlock *ledger.SnapshotBlock
	latestSnapshotBlock  *ledger.SnapshotBlock

	dataDir       string
	ledgerDirName string

	em *eventManager

	cfg         *config.Chain
	globalCfg   *config.Config
	kafkaSender *sender.KafkaSender
	trieGc      trie_gc.Collector

	saveTrieLock sync.RWMutex

	saveTrieStatus     uint8
	saveTrieStatusLock sync.Mutex

	saList *chain_cache.AdditionList
	fti    *chain_index.FilterTokenIndex
}

func NewChain(cfg *config.Config) Chain {
	chain := &chain{
		log:                  log15.New("module", "chain"),
		ledgerDirName:        "ledger",
		genesisSnapshotBlock: &GenesisSnapshotBlock,
		dataDir:              cfg.DataDir,
		cfg:                  cfg.Chain,
		globalCfg:            cfg,
	}

	if chain.cfg == nil {
		chain.cfg = &config.Chain{}
	}

	if chain.cfg.OpenFilterTokenIndex {
		var err error
		chain.fti, err = chain_index.NewFilterTokenIndex(cfg, chain)
		if err != nil {
			chain.log.Crit("NewFilterTokenIndex failed, error is "+err.Error(), "method", "NewChain")
			return nil
		}
	}

	chain.needSnapshotCache = chain_cache.NewNeedSnapshotCache(chain)
	chain.blackBlock = NewBlackBlock(chain, chain.cfg.OpenBlackBlock)

	// set ledger GenesisAccountAddress
	ledger.GenesisAccountAddress = cfg.Genesis.GenesisAccountAddress

	initGenesis(cfg.Genesis)
	return chain
}

func (c *chain) Init() {
	// Start initialize
	c.log.Info("Init chain module")

	// stateTriePool
	c.stateTriePool = NewStateTriePool(c)

	// eventManager
	c.em = newEventManager()

	// chainDb
	chainDb := chain_db.NewChainDb(filepath.Join(c.dataDir, c.ledgerDirName))
	if chainDb == nil {
		c.log.Crit("NewChain failed, db init failed", "method", "Init")
	}
	c.chainDb = chainDb

	// cache
	c.initCache()

	// saList
	var err error
	c.saList, err = chain_cache.NewAdditionList(c)
	if err != nil {
		c.log.Crit("chain_cache.NewAdditionList failed, error is "+err.Error(), "method", "Init")
	}

	// trie gc
	c.trieGc = trie_gc.NewCollector(c, c.cfg.LedgerGcRetain)

	// compressor
	compressor := compress.NewCompressor(c, c.dataDir)
	c.compressor = compressor

	// kafka sender
	if len(c.cfg.KafkaProducers) > 0 {
		var newKafkaErr error
		c.kafkaSender, newKafkaErr = sender.NewKafkaSender(c, filepath.Join(c.dataDir, "ledger_mq"))
		if newKafkaErr != nil {
			c.log.Crit("NewKafkaSender failed, error is " + newKafkaErr.Error())
		}
	}
	// Finish initialize
	c.log.Info("Chain module initialized")
}

func (c *chain) KafkaSender() *sender.KafkaSender {
	return c.kafkaSender
}

func (c *chain) checkData() bool {
	sb := c.genesisSnapshotBlock
	sb2 := SecondSnapshotBlock

	dbSb, err := c.GetSnapshotBlockByHeight(1)
	dbSb2, err2 := c.GetSnapshotBlockByHeight(2)

	if err != nil || dbSb == nil || sb.Hash != dbSb.Hash ||
		err2 != nil || dbSb2 == nil || sb2.Hash != dbSb2.Hash {
		if err != nil {
			c.log.Crit("GetSnapshotBlockByHeight failed, error is "+err.Error(), "method", "CheckAndInitDb")
		}

		if err2 != nil {
			c.log.Crit("GetSnapshotBlockByHeight(2) failed, error is "+err.Error(), "method", "CheckAndInitDb")
		}
		return false
	}

	return true
}

func (c *chain) checkForkPoints() (bool, *config.ForkPoint, error) {
	if c.globalCfg.ForkPoints == nil {
		return true, nil, nil
	}

	latestSnapshotHeight := c.GetLatestSnapshotBlock().Height

	t := reflect.TypeOf(c.globalCfg.Genesis.ForkPoints).Elem()
	v := reflect.ValueOf(c.globalCfg.Genesis.ForkPoints).Elem()

	for k := 0; k < t.NumField(); k++ {
		forkPoint := v.Field(k).Interface().(*config.ForkPoint)
		if forkPoint.Height > 0 && forkPoint.Hash != nil && forkPoint.Height <= latestSnapshotHeight {
			blockPoint, err := c.GetSnapshotBlockByHash(forkPoint.Hash)
			if err != nil {
				return false, nil, err
			}
			if blockPoint == nil {
				return false, forkPoint, nil
			}
		}
	}

	return true, nil, nil
}

func (c *chain) checkAndInitData() {
	if !c.checkData() {
		// clear data
		c.clearData()
		// init data
		c.initData()
		// init cache
		c.initCache()
	}

	noFork, forkPoint, err := c.checkForkPoints()

	if err != nil {
		c.log.Crit("checkForkPoints failed, error is "+err.Error(), "method", "checkAndInitData")
	}

	// is fork
	if !noFork {
		latestSb := c.GetLatestSnapshotBlock()
		latestHeight := latestSb.Height

		// delete to forkPoint.Height - 1
		_, _, err := c.DeleteSnapshotBlocksToHeight(forkPoint.Height)
		if err != nil {
			c.log.Crit("DeleteSnapshotBlocksToHeight failed, error is "+err.Error(), "method", "checkAndInitData")
		}

		if c.cfg.LedgerGc && forkPoint.Height+1800 < latestHeight {
			// recover trie
			c.TrieGc().Recover()
		}
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
	// Insert snapshot block
	err = c.InsertSnapshotBlock(&GenesisSnapshotBlock)
	if err != nil {
		c.log.Crit("WriteSnapshotBlock failed, error is "+err.Error(), "method", "initData")

	}

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

	// Insert second snapshot block
	err = c.InsertSnapshotBlock(&SecondSnapshotBlock)
	if err != nil {
		c.log.Crit("WriteSnapshotBlock failed, error is "+err.Error(), "method", "initData")
	}
}

func (c *chain) initCache() {
	// latestSnapshotBlock
	var getLatestBlockErr error
	c.latestSnapshotBlock, getLatestBlockErr = c.chainDb.Sc.GetLatestBlock()
	if getLatestBlockErr != nil {
		c.log.Crit("GetLatestBlock failed, error is "+getLatestBlockErr.Error(), "method", "Start")
	}

	// needSnapshotCache
	c.needSnapshotCache.Build()

	// trieNodePool
	c.trieNodePool = trie.NewTrieNodePool()
}

func (c *chain) Compressor() *compress.Compressor {
	return c.compressor
}

func (c *chain) ChainDb() *chain_db.ChainDb {
	return c.chainDb
}

func (c *chain) SaList() *chain_cache.AdditionList {
	return c.saList
}

func (c *chain) Fti() *chain_index.FilterTokenIndex {
	return c.fti
}

func (c *chain) Start() {
	// saList start
	c.saList.Start()

	// Start compress in the background
	c.log.Info("Start chain module")

	// check
	c.checkAndInitData()

	// start compressor
	c.compressor.Start()

	// start kafka sender
	if c.kafkaSender != nil {
		for _, producer := range c.cfg.KafkaProducers {
			startErr := c.kafkaSender.Start(producer.BrokerList, producer.Topic)
			if startErr != nil {
				c.log.Crit("Start kafka sender failed, error is " + startErr.Error())
			}
		}
	}

	// check trie
	if result, err := c.TrieGc().Check(); err != nil {
		panic(errors.New("c.TrieGc().Check() failed when start chain, error is " + err.Error()))
	} else if !result {
		c.TrieGc().Recover()
	}

	// trie gc
	if c.cfg.LedgerGc {
		c.TrieGc().Start()
	}

	// start build filter token index
	if c.fti != nil {
		fmt.Printf("FilterTokenIndex is being initialized...\n")
		c.fti.Start()
		fmt.Printf("FilterTokenIndex initialization complete\n")
	}

	c.log.Info("Chain module started")
}

func (c *chain) Stop() {
	// stop build filter token index
	if c.fti != nil {
		c.fti.Stop()
	}

	// saList top
	c.saList.Stop()

	// trie gc
	if c.cfg.LedgerGc {
		c.TrieGc().Stop()
	}
	// Stop compress
	c.log.Info("Stop chain module")

	// stop compressor
	c.compressor.Stop()

	// stop kafka sender
	if c.kafkaSender != nil {
		c.kafkaSender.StopAll()
	}

	c.log.Info("Chain module stopped")
}

func (c *chain) Destroy() {
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

	// kafka sender
	c.kafkaSender = nil

	c.log.Info("Chain module destroyed")
}
func (c *chain) TrieGc() trie_gc.Collector {
	return c.trieGc
}

func (c *chain) TrieDb() *leveldb.DB {
	return c.ChainDb().Db()
}
