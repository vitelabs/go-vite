package chain

import (
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/sender"
	"github.com/vitelabs/go-vite/chain/trie_gc"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"os"
	"path/filepath"
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

	needSnapshotCache *NeedSnapshotCache

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

	*fork
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

	chain.blackBlock = NewBlackBlock(chain, chain.cfg.OpenBlackBlock)

	initGenesis(chain.readGenesis(chain.cfg.GenesisFile))

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
			c.log.Error("GetSnapshotBlockByHeight failed, error is "+err.Error(), "method", "CheckAndInitDb")
		}

		if err2 != nil {
			c.log.Error("GetSnapshotBlockByHeight(2) failed, error is "+err.Error(), "method", "CheckAndInitDb")
		}
		return false
	}
	return true
}
func (c *chain) checkAndInitData() {
	if !c.checkData() {
		// clear data
		c.clearData()
		// init data
		c.initData()
		// init cache
		c.initCache()
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
	unconfirmedSubLedger, getSubLedgerErr := c.getUnConfirmedSubLedger()
	if getSubLedgerErr != nil {
		c.log.Crit("getUnConfirmedSubLedger failed, error is "+getSubLedgerErr.Error(), "method", "Start")
	}
	c.needSnapshotCache = NewNeedSnapshotContent(c, unconfirmedSubLedger)

	// trieNodePool
	c.trieNodePool = trie.NewTrieNodePool()
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

	// trie gc
	if c.cfg.LedgerGc {
		c.trieGc.Start()
	}

	c.log.Info("Chain module started")
}

func (c *chain) Stop() {
	// trie gc
	if c.cfg.LedgerGc {
		c.trieGc.Stop()
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

func (c *chain) readGenesis(genesisPath string) *GenesisConfig {
	defaultGenesisAccountAddress, _ := types.HexToAddress("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")
	var defaultBlockProducers []types.Address
	addrStrList := []string{
		"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
		"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08",
		"vite_1630f8c0cf5eda3ce64bd49a0523b826f67b19a33bc2a5dcfb",
		"vite_1b1dfa00323aea69465366d839703547fec5359d6c795c8cef",
		"vite_27a258dd1ed0ce0de3f4abd019adacd1b4b163b879389d3eca",
		"vite_31a02e4f4b536e2d6d9bde23910cdffe72d3369ef6fe9b9239",
		"vite_383fedcbd5e3f52196a4e8a1392ed3ddc4d4360e4da9b8494e",
		"vite_41ba695ff63caafd5460dcf914387e95ca3a900f708ac91f06",
		"vite_545c8e4c74e7bb6911165e34cbfb83bc513bde3623b342d988",
		"vite_5a1b5ece654138d035bdd9873c1892fb5817548aac2072992e",
		"vite_70cfd586185e552635d11f398232344f97fc524fa15952006d",
		"vite_76df2a0560694933d764497e1b9b11f9ffa1524b170f55dda0",
		"vite_7b76ca2433c7ddb5a5fa315ca861e861d432b8b05232526767",
		"vite_7caaee1d51abad4047a58f629f3e8e591247250dad8525998a",
		"vite_826a1ab4c85062b239879544dc6b67e3b5ce32d0a1eba21461",
		"vite_89007189ad81c6ee5cdcdc2600a0f0b6846e0a1aa9a58e5410",
		"vite_9abcb7324b8d9029e4f9effe76f7336bfd28ed33cb5b877c8d",
		"vite_af60cf485b6cc2280a12faac6beccfef149597ea518696dcf3",
		"vite_c1090802f735dfc279a6c24aacff0e3e4c727934e547c24e5e",
		"vite_c10ae7a14649800b85a7eaaa8bd98c99388712412b41908cc0",
		"vite_d45ac37f6fcdb1c362a33abae4a7d324a028aa49aeea7e01cb",
		"vite_d8974670af8e1f3c4378d01d457be640c58644bc0fa87e3c30",
		"vite_e289d98f33c3ef5f1b41048c2cb8b389142f033d1df9383818",
		"vite_f53dcf7d40b582cd4b806d2579c6dd7b0b131b96c2b2ab5218",
		"vite_fac06662d84a7bea269265e78ea2d9151921ba2fae97595608",
	}

	for _, addrStr := range addrStrList {
		addr, _ := types.HexToAddress(addrStr)
		defaultBlockProducers = append(defaultBlockProducers, addr)
	}

	defaultSnapshotConsensusGroup := ConsensusGroupInfo{
		NodeCount:           25,
		Interval:            1,
		PerCount:            3,
		RandCount:           2,
		RandRank:            100,
		CountingTokenId:     ledger.ViteTokenId,
		RegisterConditionId: 1,
		RegisterConditionParam: ConditionRegisterData{
			PledgeAmount: new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
			PledgeHeight: uint64(3600 * 24 * 90),
			PledgeToken:  ledger.ViteTokenId,
		},
		VoteConditionId: 1,
		Owner:           defaultGenesisAccountAddress,
		PledgeAmount:    big.NewInt(0),
		WithdrawHeight:  1,
	}
	defaultCommonConsensusGroup := ConsensusGroupInfo{
		NodeCount:           25,
		Interval:            3,
		PerCount:            1,
		RandCount:           2,
		RandRank:            100,
		CountingTokenId:     ledger.ViteTokenId,
		RegisterConditionId: 1,
		RegisterConditionParam: ConditionRegisterData{
			PledgeAmount: new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
			PledgeHeight: uint64(3600 * 24 * 90),
			PledgeToken:  ledger.ViteTokenId,
		},
		VoteConditionId: 1,
		Owner:           defaultGenesisAccountAddress,
		PledgeAmount:    big.NewInt(0),
		WithdrawHeight:  1,
	}

	config := &GenesisConfig{
		GenesisAccountAddress: defaultGenesisAccountAddress,
		BlockProducers:        defaultBlockProducers,
	}

	if len(genesisPath) > 0 {
		file, err := os.Open(genesisPath)
		if err != nil {
			c.log.Crit(fmt.Sprintf("Failed to read genesis file: %v", err), "method", "readGenesis")
		}
		defer file.Close()

		config = new(GenesisConfig)
		if err := json.NewDecoder(file).Decode(config); err != nil {
			c.log.Crit(fmt.Sprintf("invalid genesis file: %v", err), "method", "readGenesis")
		}
	}

	if config.SnapshotConsensusGroup == nil {
		config.SnapshotConsensusGroup = &defaultSnapshotConsensusGroup
	}

	if config.CommonConsensusGroup == nil {
		config.CommonConsensusGroup = &defaultCommonConsensusGroup
	}

	// set fork
	config.Fork = NewFork(config)
	c.fork = config.Fork

	// hack, will be fix
	ledger.GenesisAccountAddress = config.GenesisAccountAddress

	return config
}
