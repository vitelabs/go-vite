package pool

import (
	"sync"
	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/verifier"
	"github.com/vitelabs/go-vite/interval/version"
)

type snapshotPool struct {
	BCPool
	rwMu *sync.RWMutex
	//consensus consensus.AccountsConsensus
	closed chan struct{}
	wg     sync.WaitGroup
	pool   *pool
}

func newSnapshotPool(name string, v *version.Version) *snapshotPool {
	pool := &snapshotPool{}
	pool.Id = name
	pool.version = v
	pool.closed = make(chan struct{})
	return pool
}

func (spl *snapshotPool) init(rw chainRw,
	verifier verifier.Verifier,
	syncer *fetcher,
	rwMu *sync.RWMutex,
	pool *pool) {
	spl.rwMu = rwMu
	//spl.consensus = accountsConsensus
	spl.pool = pool
	spl.BCPool.init(rw, verifier, syncer)
}

func (spl *snapshotPool) loopCheckFork() {
	defer spl.wg.Done()
	for {
		select {
		case <-spl.closed:
			return
		default:
			spl.checkFork()
			// check fork every 2 sec.
			time.Sleep(2 * time.Second)
		}
	}
}

func (spl *snapshotPool) checkFork() {
	longest := spl.LongestChain()
	current := spl.CurrentChain()
	if longest.ChainId() == current.ChainId() {
		return
	}
	spl.snapshotFork(longest, current)

}

func (spl *snapshotPool) snapshotFork(longest Chain, current Chain) {
	log.Warn("[try]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())
	spl.rwMu.Lock()
	defer spl.rwMu.Unlock()
	log.Warn("[lock]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())

	k, f, err := spl.getForkPoint(longest, current)
	if err != nil {
		log.Error("get snapshot forkPoint err. err:%v", err)
		return
	}
	forkPoint := f.(*common.SnapshotBlock)
	keyPoint := k.(*common.SnapshotBlock)

	startAcs, endAcs := spl.getUnlockAccountSnapshot(forkPoint)

	spl.pool.Rollback(keyPoint.Height())
	err = spl.pool.UnLockAccounts(startAcs, endAcs)
	if err != nil {
		log.Error("unlock accounts fail. err:%v", err)
		return
	}
	err = spl.pool.ForkAccounts(keyPoint, forkPoint)
	if err != nil {
		log.Error("rollback accounts fail. err:%v", err)
		return
	}
	err = spl.Rollback(forkPoint.Height(), forkPoint.Hash())
	if err != nil {
		log.Error("rollback snapshot fail. err:%v", err)
		return
	}
	err = spl.CurrentModifyToChain(longest)
	if err != nil {
		log.Error("snapshot modify current fail. err:%v", err)
		return
	}
	spl.version.Inc()
}

func (spl *snapshotPool) loop() {
	defer spl.wg.Done()
	for {
		select {
		case <-spl.closed:
			return
		default:
			spl.loopGenSnippetChains()
			spl.loopAppendChains()
			spl.loopFetchForSnippets()
			spl.loopCheckCurrentInsert()
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (spl *snapshotPool) loopCheckCurrentInsert() {
	if spl.chainpool.current.size() == 0 {
		return
	}
	spl.rwMu.RLock()
	defer spl.rwMu.RUnlock()
	spl.snapshotTryInsert()
}

func (spl *snapshotPool) snapshotTryInsert() {
	pool := spl.chainpool
	current := pool.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
		block := wrapper.block

		if !wrapper.checkForkVersion() {
			wrapper.reset()
		}
		stat := pool.verifier.VerifyReferred(block)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			continue
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:
			spl.insertVerifyPending(block, stat)
			break L
		case verifier.FAIL:
			log.Error("snapshot verify fail. block info:hash[%s],height[%d], %s",
				result, block.Hash(), block.Height(), stat.ErrMsg())
			spl.insertVerifyFail(block, stat)
			break L
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := pool.writeToChain(current, wrapper)
				if err != nil {
					log.Error("insert snapshot chain fail. block info:hash[%s],height[%d], err:%v",
						block.Hash(), block.Height(), err)
					break L
				} else {
					spl.blockpool.afterInsert(wrapper)
				}
			} else {
				break L
			}
		default:
			log.Fatal("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			break L
		}
	}
}
func (spl *snapshotPool) Start() {
	spl.wg.Add(1)
	go spl.loop()
	spl.wg.Add(1)
	go spl.loopCheckFork()
	log.Info("snapshot_pool[%s] started.", spl.Id)
}
func (spl *snapshotPool) Stop() {
	close(spl.closed)
	spl.wg.Wait()
	log.Info("snapshot_pool[%s] stopped.", spl.Id)
}

func (spl *snapshotPool) insertVerifyFail(b common.Block, s verifier.BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)
	stat := s.(*verifier.SnapshotBlockVerifyStat)
	results := stat.Results()

	for _, account := range block.Accounts {
		result := results[account.Addr]
		if result == verifier.FAIL {
			spl.pool.ForkAccountTo(account)
		}
	}
	spl.version.Inc()
}

func (spl *snapshotPool) insertVerifyPending(b common.Block, s verifier.BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)
	stat := s.(*verifier.SnapshotBlockVerifyStat)
	results := stat.Results()

	for _, account := range block.Accounts {
		result := results[account.Addr]
		if result == verifier.PENDING {
			spl.pool.PendingAccountTo(account)
		}
	}
}

func (spl *snapshotPool) getUnlockAccountSnapshot(block *common.SnapshotBlock) (map[string]*common.SnapshotPoint, map[string]*common.SnapshotPoint) {
	h := spl.chainpool.diskChain.Head()
	head := h.(*common.SnapshotBlock)
	startAcs := make(map[string]*common.SnapshotPoint)
	endAcs := make(map[string]*common.SnapshotPoint)

	spl.accounts(startAcs, endAcs, head)
	for i := head.Height() - 1; i > block.Height(); i-- {
		b := spl.chainpool.diskChain.getBlock(i, false)
		if b != nil {
			block := b.block.(*common.SnapshotBlock)
			spl.accounts(startAcs, endAcs, block)
		}
	}
	return startAcs, endAcs
}

func (spl *snapshotPool) accounts(start map[string]*common.SnapshotPoint, end map[string]*common.SnapshotPoint, block *common.SnapshotBlock) {
	hs := block.Accounts
	for _, v := range hs {
		point := &common.SnapshotPoint{SnapshotHeight: block.Height(), SnapshotHash: block.Hash(), AccountHeight: v.Height, AccountHash: v.Hash}
		s := start[v.Addr]
		if s == nil {
			start[v.Addr] = point
		}
		end[v.Addr] = point
	}
}
