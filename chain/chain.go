package chain

import (
	"fmt"
	"github.com/olebedev/emitter"
	"github.com/vitelabs/go-vite/common/fork"

	"github.com/vitelabs/go-vite/chain/plugins"

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
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common"
)

const (
	stop  = 0
	start = 1
)

type chain struct {
	genesisCfg *config.Genesis
	chainCfg   *config.Chain

	genesisSnapshotBlock    *ledger.SnapshotBlock
	genesisAccountBlocks    []*vm_db.VmAccountBlock
	genesisAccountBlockHash map[types.Hash]struct{}

	dataDir   string
	chainDir  string
	consensus Consensus

	log log15.Logger

	em *eventManager

	emitter *emitter.Emitter

	cache *chain_cache.Cache

	metaDB *leveldb.DB

	indexDB *chain_index.IndexDB

	blockDB *chain_block.BlockDB

	stateDB *chain_state.StateDB

	syncCache interfaces.SyncCache

	flusher *chain_flusher.Flusher

	flushMu sync.RWMutex

	plugins *chain_plugins.Plugins

	status uint32

	forkActiveCheckPoint fork.ForkPointItem
	forkActiveCache      fork.ForkPointList
}

/*
 * Init chain config
 */
func NewChain(dir string, chainCfg *config.Chain, genesisCfg *config.Genesis) *chain {
	if chainCfg == nil {
		chainCfg = defaultConfig()
	}

	if !fork.IsInitForkPoint() {
		fork.SetForkPoints(genesisCfg.ForkPoints)
	}

	c := &chain{
		genesisCfg: genesisCfg,
		dataDir:    dir,
		chainDir:   path.Join(dir, "ledger"),

		log: log15.New("module", "chain"),

		emitter:  emitter.New(10),
		chainCfg: chainCfg,
	}

	// set leaf fork point
	forkActiveCheckPoint := fork.GetLeafForkPoint()
	if forkActiveCheckPoint == nil {
		panic("LeafFork is not existed")
	}

	c.forkActiveCheckPoint = *forkActiveCheckPoint

	// set active fork
	if !fork.IsInitActiveChecker() {
		fork.SetActiveChecker(c)
	}

	c.em = newEventManager(c)
	c.emitter.Use("*", emitter.Sync)

	c.genesisAccountBlocks = chain_genesis.NewGenesisAccountBlocks(genesisCfg)
	c.genesisSnapshotBlock = chain_genesis.NewGenesisSnapshotBlock(c.genesisAccountBlocks)

	c.genesisAccountBlockHash = chain_genesis.VmBlocksToHashMap(c.genesisAccountBlocks)

	return c
}

/*
 * 1. Check and init ledger (check genesis block)
 * 2. Init indexDB
 * 3. Init stateDB
 * 4. Init blockDB
 * 5. Init cache(indexDB cache, stateDB cache, blockDB cache, syncCache)
 */
func (c *chain) Init() error {
	c.log.Info("Begin initializing", "method", "Init")

	// init db
	if err := c.newDbAndRecover(); err != nil {
		return err
	}

	// check ledger
	status, err := c.checkAndInitData()
	if err != nil {
		return err
	}

	// ledger is invalid
	if status != chain_genesis.LedgerValid {
		return errors.New(fmt.Sprintf("The genesis state is incorrect. You can fix the problem by removing the database manually."+
			"The directory of database is %s.", c.chainDir))
	}

	// init cache
	if err := c.initCache(); err != nil {
		return err
	}

	// init fork active
	if err := c.initActiveFork(); err != nil {
		return err
	}

	// check fork points and rollback
	if err := c.checkForkPointsAndRollback(); err != nil {
		return err
	}

	// reconstruct the plugins
	/*	if c.chainCfg.OpenPlugins {
			c.plugins.BuildPluginsDb(c.flusher)
		}
	*/
	c.log.Info("Complete initialization", "method", "Init")

	return nil
}

func (c *chain) Start() error {
	if !atomic.CompareAndSwapUint32(&c.status, stop, start) {
		return nil
	}

	c.flusher.Start()
	c.log.Info("Start flusher", "method", "Start")

	return nil
}

func (c *chain) Stop() error {
	if !atomic.CompareAndSwapUint32(&c.status, start, stop) {
		return nil
	}

	c.flusher.Stop()

	c.log.Info("Stop flusher", "method", "Stop")
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

	if err := c.syncCache.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.syncCache.Close failed, error is %s", err))
		c.log.Error(cErr.Error(), "method", "Close")
		return cErr
	}
	c.log.Info("Close syncCache", "method", "Close")

	c.flusher = nil
	c.cache = nil
	c.stateDB = nil
	c.indexDB = nil
	c.blockDB = nil
	c.syncCache = nil

	c.log.Info("Complete destruction", "method", "Close")

	return nil
}

func (c *chain) Plugins() *chain_plugins.Plugins {
	return c.plugins
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
	c.log.Info("Start set consensus", "method", "SetConsensus")
	c.consensus = cs

	if err := c.stateDB.SetConsensus(cs); err != nil {
		c.log.Crit(fmt.Sprintf("c.stateDB.SetConsensus failed. Error: %s", err.Error()), "method", "SetConsensus")
	}
	c.log.Info("set consensus finished", "method", "SetConsensus")
}

func (c *chain) newDbAndRecover() error {
	var err error
	// new metaDB
	c.metaDB, err = c.NewDb("chain_meta")
	if err != nil {
		c.log.Error(fmt.Sprintf("new meta db failed, error is %s, chainDir is %s", err, c.chainDir), "method", "newDbAndRecover")
		return err
	}

	// new ledger db
	if c.indexDB, err = chain_index.NewIndexDB(c.chainDir, c); err != nil {
		c.log.Error(fmt.Sprintf("chain_index.NewIndexDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "newDbAndRecover")
		return err
	}

	// new block db
	if c.blockDB, err = chain_block.NewBlockDB(c.chainDir); err != nil {
		c.log.Error(fmt.Sprintf("chain_block.NewBlockDB failed, error is %s, chainDir is %s", err, c.chainDir), "method", "newDbAndRecover")
		return err
	}

	// new state db
	if c.stateDB, err = chain_state.NewStateDB(c, c.chainCfg, c.chainDir); err != nil {
		cErr := errors.New(fmt.Sprintf("chain_cache.NewStateDB failed, error is %s", err))

		c.log.Error(cErr.Error(), "method", "newDbAndRecover")
		return err
	}

	// init plugins
	if c.chainCfg.OpenPlugins {
		var err error
		if c.plugins, err = chain_plugins.NewPlugins(c.chainDir, c); err != nil {
			cErr := errors.New(fmt.Sprintf("chain_plugins.NewPlugins failed. Error: %s", err))
			c.log.Error(cErr.Error(), "method", "newDbAndRecover")
			return cErr
		}
		c.Register(c.plugins)
	}

	// new flusher
	stores := []chain_flusher.Storage{c.blockDB, c.stateDB.Store(), c.stateDB.RedoStore(), c.indexDB.Store()}
	if c.chainCfg.OpenPlugins {
		stores = append(stores, c.plugins.Store())
	}
	if c.flusher, err = chain_flusher.NewFlusher(stores, &c.flushMu, c.chainDir); err != nil {
		cErr := errors.New(fmt.Sprintf("chain_flusher.NewFlusher failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "newDbAndRecover")
		return cErr
	}

	// flusher check and recover
	if err := c.flusher.Recover(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.flusher.Recover failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "newDbAndRecover")
		return cErr
	}

	// new cache
	if c.cache, err = chain_cache.NewCache(c); err != nil {
		cErr := errors.New(fmt.Sprintf("chain_cache.NewCache failed, error is %s", err))

		c.log.Error(cErr.Error(), "method", "checkAndInitData")
		return cErr
	}

	return nil
}

func (c *chain) checkAndInitData() (byte, error) {
	// check ledger
	status, err := chain_genesis.CheckLedger(c, c.genesisSnapshotBlock, c.genesisAccountBlocks)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("chain_genesis.CheckLedger failed, error is %s, chainDir is %s", err, c.chainDir))

		c.log.Error(cErr.Error(), "method", "checkAndInitData")
		return status, err
	}

	if status == chain_genesis.LedgerInvalid {
		return status, nil
	}

	if status == chain_genesis.LedgerEmpty {
		if err = chain_genesis.InitLedger(c, c.genesisSnapshotBlock, c.genesisAccountBlocks); err != nil {
			cErr := errors.New(fmt.Sprintf("chain_genesis.InitLedger failed, error is %s", err))
			c.log.Error(cErr.Error(), "method", "checkAndInitData")
			return chain_genesis.LedgerInvalid, err
		}

		status = chain_genesis.LedgerValid
	}

	return status, nil
}

func (c *chain) checkForkPointsAndRollback() error {
	forkPointList := fork.GetActiveForkPointList()

	// check
	var rollbackForkPoint *fork.ForkPointItem
	for i := len(forkPointList) - 1; i >= 0; i-- {
		forkPoint := forkPointList[i]
		sb, err := c.GetSnapshotBlockByHeight(forkPoint.Height)
		if err != nil {
			return err
		}

		if sb == nil {
			continue
		}

		if sb.ComputeHash() == sb.Hash {
			break
		}
		rollbackForkPoint = forkPoint
	}

	// rollback
	if rollbackForkPoint != nil {
		if _, err := c.DeleteSnapshotBlocksToHeight(rollbackForkPoint.Height); err != nil {
			return err
		}
	}

	return nil
}

func (c *chain) initCache() error {

	// init cache
	if err := c.cache.Init(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cache.Init failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "initCache")
		return cErr
	}

	// init state db cache
	if err := c.stateDB.Init(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Init failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "initCache")
		return cErr
	}

	// init index db cache
	if err := c.indexDB.Init(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Init failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "initCache")
		return cErr
	}

	// init sync cache
	var err error
	c.syncCache, err = sync_cache.NewSyncCache(path.Join(c.chainDir, "sync_cache"))
	if err != nil {
		cErr := errors.New(fmt.Sprintf("sync_cache.NewSyncCache failed. Error: %s", err))
		c.log.Error(cErr.Error(), "method", "initCache")
		return cErr
	}

	return nil
}

func (c *chain) closeAndCleanData() error {
	var err error
	// close blockDB
	if err = c.blockDB.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.blockDB.Close failed. Error: %s", err))

		c.log.Error(cErr.Error(), "method", "closeAndCleanData")
		return err
	}

	// close indexDB
	if err = c.indexDB.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.indexDB.Close failed. Error: %s", err))

		c.log.Error(cErr.Error(), "method", "closeAndCleanData")
		return err
	}

	// close stateDB
	if err = c.stateDB.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.Close failed. Error: %s", err))

		c.log.Error(cErr.Error(), "method", "closeAndCleanData")
		return err
	}

	// close flusher
	if err = c.flusher.Close(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.flusher.Close failed. Error: %s", err))

		c.log.Error(cErr.Error(), "method", "closeAndCleanData")
		return err
	}

	// close plugins
	if c.chainCfg.OpenPlugins {
		if err = c.plugins.Close(); err != nil {
			cErr := errors.New(fmt.Sprintf("c.plugins.Close failed. Error: %s", err))

			c.log.Error(cErr.Error(), "method", "closeAndCleanData")
			return err
		}
	}

	// clean all data
	if err = c.cleanAllData(); err != nil {
		cErr := errors.New(fmt.Sprintf("c.cleanAllData failed. Error: %s", err))

		c.log.Error(cErr.Error(), "method", "closeAndCleanData")
		return err
	}
	return nil
}

func (c *chain) cleanAllData() error {
	return os.RemoveAll(c.chainDir)
}

func defaultConfig() *config.Chain {
	return &config.Chain{
		LedgerGc:       true,
		LedgerGcRetain: 24 * 3600,
		OpenPlugins:    false,
	}
}

func (c *chain) DBs() (*chain_index.IndexDB, *chain_block.BlockDB, *chain_state.StateDB) {
	return c.indexDB, c.blockDB, c.stateDB
}

func (c *chain) Flusher() *chain_flusher.Flusher {
	return c.flusher
}

func (c *chain) ResetLog(dir string, lvl string) {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	path := filepath.Join(dir, "chain_logs", time.Now().Format("2006-01-02T15-04"))
	filename := filepath.Join(path, "chain.log")

	h := log15.LvlFilterHandler(logLevel, log15.StreamHandler(common.MakeDefaultLogger(filename), log15.LogfmtFormat()))

	c.log.SetHandler(
		h,
	)

	c.blockDB.SetLog(h)
}

func (c *chain) GetStatus() []interfaces.DBStatus {
	var statusList = make([]interfaces.DBStatus, 0)

	statusList = append(statusList, c.cache.GetStatus()...)
	statusList = append(statusList, c.indexDB.GetStatus()...)
	statusList = append(statusList, c.blockDB.GetStatus()...)
	statusList = append(statusList, c.stateDB.GetStatus()...)

	return statusList
}

func (c *chain) SetCacheLevelForConsensus(level uint32) {
	c.stateDB.SetCacheLevelForConsensus(level)
}

func (c *chain) StopWrite() {
	c.flushMu.Lock()
}

func (c *chain) RecoverWrite() {
	c.flushMu.Unlock()
}
