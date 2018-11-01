package pool

import (
	"sync"
	"time"

	"fmt"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
)

type snapshotPool struct {
	BCPool
	//rwMu *sync.RWMutex
	//consensus consensus.AccountsConsensus
	closed chan struct{}
	wg     sync.WaitGroup
	pool   *pool
	rw     *snapshotCh
	v      *snapshotVerifier
	f      *snapshotSyncer
}

func newSnapshotPoolBlock(block *ledger.SnapshotBlock, version *ForkVersion, source types.BlockSource) *snapshotPoolBlock {
	return &snapshotPoolBlock{block: block, forkBlock: *newForkBlock(version, source)}
}

type snapshotPoolBlock struct {
	forkBlock
	block *ledger.SnapshotBlock
}

func (self *snapshotPoolBlock) Height() uint64 {
	return self.block.Height
}

func (self *snapshotPoolBlock) Hash() types.Hash {
	return self.block.Hash
}

func (self *snapshotPoolBlock) PrevHash() types.Hash {
	return self.block.PrevHash
}

func (self *snapshotPoolBlock) Source() types.BlockSource {
	return self.source
}

func newSnapshotPool(
	name string,
	version *ForkVersion,
	v *snapshotVerifier,
	f *snapshotSyncer,
	rw *snapshotCh,
	log log15.Logger,
) *snapshotPool {
	pool := &snapshotPool{}
	pool.Id = name
	pool.version = version
	pool.rw = rw
	pool.v = v
	pool.f = f
	pool.log = log.New("snapshotPool", name)
	return pool
}

func (self *snapshotPool) init(
	tools *tools,
	pool *pool) {
	//self.consensus = accountsConsensus
	self.pool = pool
	self.BCPool.init(tools)
}

func (self *snapshotPool) loopCheckFork() {
	// recover logic
	defer func() {
		if err := recover(); err != nil {
			var e error
			switch t := err.(type) {
			case error:
				e = errors.WithStack(t)
			case string:
				e = errors.New(t)
			default:
				e = errors.Errorf("unknown type, %+v", err)
			}

			self.log.Error("loopCheckFork start recover", "err", err, "withstack", fmt.Sprintf("%+v", e))
			fmt.Printf("%+v", e)
			defer self.log.Warn("loopCheckFork end recover.")
			self.pool.Lock()
			defer self.pool.UnLock()
			self.initPool()
			if self.rstat.inc() {
				common.Go(self.loopCheckFork)
			} else {
				panic(e)
			}

			self.pool.version.Inc()
		}
	}()
	self.wg.Add(1)
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.checkFork()
			// check fork every 2 sec.
			time.Sleep(2 * time.Second)
		}
	}
}

func (self *snapshotPool) checkFork() {
	longest := self.LongestChain()
	current := self.CurrentChain()
	if longest.ChainId() == current.ChainId() {
		return
	}
	monitor.LogEvent("pool", "snapshotFork")
	err := self.snapshotFork(longest, current)
	if err != nil {
		self.log.Error("checkFork", "err", err)
	}
}

func (self *snapshotPool) snapshotFork(longest *forkedChain, current *forkedChain) error {
	defer monitor.LogTime("pool", "snapshotFork", time.Now())
	self.log.Warn("[try]snapshot chain start fork.", "longest", longest.ChainId(), "current", current.ChainId(),
		"longestTailHeight", longest.tailHeight, "longestHeadHeight", longest.headHeight, "currentTailHeight", current.tailHeight, "currentHeadHeight", current.headHeight)
	self.pool.Lock()
	defer self.pool.UnLock()
	self.log.Warn("[lock]snapshot chain start fork.", "longest", longest.ChainId(), "current", current.ChainId())

	k, forked, err := self.getForkPoint(longest, current)
	if err != nil {
		self.log.Error("get snapshot forkPoint err.", "err", err)
		return err
	}
	if k == nil {
		self.log.Error("keypoint is empty.", "forked", forked.Height())
		return errors.New("key point is nil.")
	}
	keyPoint := k.(*snapshotPoolBlock)
	self.log.Info("fork point", "height", keyPoint.Height(), "hash", keyPoint.Hash())

	snapshots, accounts, e := self.rw.delToHeight(keyPoint.block.Height)
	if e != nil {
		return e
	}

	if len(snapshots) > 0 {
		err = self.rollbackCurrent(snapshots)
		if err != nil {
			return err
		}
	}

	if len(accounts) > 0 {
		err = self.pool.ForkAccounts(accounts)
		if err != nil {
			return err
		}
	}

	self.log.Debug("snapshotFork longest modify", "id", longest.id(), "TailHeight", longest.tailHeight, "HeadHeight", longest.headHeight)
	err = self.CurrentModifyToChain(longest, &ledger.HashHeight{Hash: forked.Hash(), Height: forked.Height()})
	if err != nil {
		return err
	}
	self.version.Inc()
	return nil
}

func (self *snapshotPool) loop() {
	// recover logic
	defer func() {
		if err := recover(); err != nil {
			var e error
			switch t := err.(type) {
			case error:
				e = errors.WithStack(t)
			case string:
				e = errors.New(t)
			default:
				e = errors.Errorf("unknown type, %+v", err)
			}

			self.log.Error("snapshot loop start recover", "err", err, "withstack", fmt.Sprintf("%+v", e))
			fmt.Printf("%+v", e)
			defer self.log.Warn("snapshot loop end recover.")
			self.pool.Lock()
			defer self.pool.UnLock()
			self.initPool()
			if self.rstat.inc() {
				common.Go(self.loop)
			} else {
				panic(e)
			}
			self.pool.version.Inc()
		}
	}()

	self.wg.Add(1)
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.loopAll()
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (self *snapshotPool) loopAll() {
	self.pool.RLock()
	defer self.pool.RUnLock()
	self.loopGenSnippetChains()
	self.loopAppendChains()
	self.loopFetchForSnippets()
	self.loopCheckCurrentInsert()
}

func (self *snapshotPool) loopCheckCurrentInsert() {
	if self.chainpool.current.size() == 0 {
		return
	}
	stat, block := self.snapshotTryInsert()

	if stat != nil {
		if stat.verifyResult() == verifier.FAIL {
			self.insertVerifyFail(block.(*snapshotPoolBlock), stat)
		} else if stat.verifyResult() == verifier.PENDING {
			self.insertVerifyPending(block.(*snapshotPoolBlock), stat)
		}
	}
}

func (self *snapshotPool) snapshotTryInsert() (*poolSnapshotVerifyStat, commonBlock) {
	self.rMu.Lock()
	defer self.rMu.Unlock()

	pool := self.chainpool
	current := pool.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		block := current.getBlock(i, false)

		if !block.checkForkVersion() {
			block.resetForkVersion()
		}
		stat := self.v.verifySnapshot(block.(*snapshotPoolBlock))
		if !block.checkForkVersion() {
			block.resetForkVersion()
			continue
		}
		result := stat.verifyResult()
		switch result {
		case verifier.PENDING:
			return stat, block
		case verifier.FAIL:
			self.log.Error("snapshot verify fail."+stat.errMsg(),
				"hash", block.Hash(), "height", block.Height())
			return stat, block
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := pool.writeToChain(current, block)
				if err != nil {
					self.log.Error("insert snapshot chain fail.",
						"hash", block.Hash(), "height", block.Height(), "err", err)
					break L
				} else {
					self.blockpool.afterInsert(block)
				}
			} else {
				break L
			}
		default:
			self.log.Crit("Unexpected things happened. ",
				"result", result, "hash", block.Hash(), "height", block.Height())
			break L
		}
	}
	return nil, nil
}
func (self *snapshotPool) Start() {
	self.closed = make(chan struct{})
	common.Go(self.loop)
	common.Go(self.loopCheckFork)
	self.log.Info("snapshot_pool started.")
}
func (self *snapshotPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	self.log.Info("snapshot_pool stopped.")
}

func (self *snapshotPool) insertVerifyFail(b *snapshotPoolBlock, stat *poolSnapshotVerifyStat) {
	block := b.block
	results := stat.results

	accounts := make(map[types.Address]*ledger.HashHeight)

	for k, account := range block.SnapshotContent {
		result := results[k]
		if result == verifier.FAIL {
			accounts[k] = account
		}
	}

	if len(accounts) > 0 {
		self.log.Debug("insertVerifyFail", "accountsLen", len(accounts))
		monitor.LogEventNum("pool", "snapshotFailFork", len(accounts))
		self.forkAccounts(b, accounts, b.Height())
	}
}

func (self *snapshotPool) forkAccounts(b *snapshotPoolBlock, accounts map[types.Address]*ledger.HashHeight, sHeight uint64) {
	self.pool.Lock()
	defer self.pool.UnLock()

	for k, v := range accounts {
		self.log.Debug("forkAccounts", "Addr", k.String(), "Height", v.Height, "Hash", v.Hash)
		err := self.pool.ForkAccountTo(k, v, sHeight)
		if err != nil {
			self.log.Error("forkaccountTo err", "err", err)
		}
	}

	self.version.Inc()
}

func (self *snapshotPool) insertVerifyPending(b *snapshotPoolBlock, stat *poolSnapshotVerifyStat) {
	block := b.block

	results := stat.results

	accounts := make(map[types.Address]*ledger.HashHeight)

	for k, account := range block.SnapshotContent {
		result := results[k]
		if result == verifier.PENDING {
			monitor.LogEvent("pool", "snapshotPending")
			self.log.Debug("pending for account.", "addr", k.String(), "height", account.Height, "hash", account.Hash)
			hashH, e := self.pool.PendingAccountTo(k, account, b.Height())
			if e != nil {
				self.log.Error("pending for account fail.", "err", e, "address", k, "hashH", account)
			}
			if hashH != nil {
				accounts[k] = account
			}
		}
	}
	if len(accounts) > 0 {
		monitor.LogEventNum("pool", "snapshotPendingFork", len(accounts))
		self.forkAccounts(b, accounts, b.Height())
	}
}

func (self *snapshotPool) AddDirectBlock(block *snapshotPoolBlock) error {
	self.rMu.Lock()
	defer self.rMu.Unlock()

	stat := self.v.verifySnapshot(block)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		return errors.New("pending for something")
	case verifier.FAIL:
		return errors.New(stat.errMsg())
	case verifier.SUCCESS:
		err := self.chainpool.diskChain.rw.insertBlock(block)
		if err != nil {
			return err
		}
		head := self.chainpool.diskChain.Head()
		self.chainpool.insertNotify(head)
		return nil
	default:
		self.log.Crit("verify unexpected.")
		return errors.New("verify unexpected")
	}
}
