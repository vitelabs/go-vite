package model

import (
	"container/list"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"sync"
	"time"
)

var (
	fullCacheExpireTime   = 2 * time.Minute
	simpleCacheExpireTime = 20 * time.Minute
)

// obtaining the account info from cache or db and manage the cache lifecycle
type OnroadBlocksPool struct {
	dbAccess *UAccess

	fullCache          *sync.Map // map[types.Address]*onroadBlocksCache
	fullCacheDeadTimer *sync.Map // map[types.Address]*time.Timer

	simpleCache          *sync.Map // map[types.Address]*OnroadAccountInfo
	simpleCacheDeadTimer *sync.Map //map[types.Address]*time.Timer

	newCommonTxListener   map[types.Address]func()
	commonTxListenerMutex sync.RWMutex

	newContractListener   map[types.Gid]func(address types.Address)
	contractListenerMutex sync.RWMutex

	log log15.Logger
}

func NewOnroadBlocksPool(dbAccess *UAccess) *OnroadBlocksPool {
	return &OnroadBlocksPool{
		dbAccess:             dbAccess,
		fullCache:            &sync.Map{},
		fullCacheDeadTimer:   &sync.Map{},
		simpleCache:          &sync.Map{},
		simpleCacheDeadTimer: &sync.Map{},
		newCommonTxListener:  make(map[types.Address]func()),
		newContractListener:  make(map[types.Gid]func(address types.Address)),
		log:                  log15.New("onroad", "OnroadBlocksPool"),
	}
}

func (p *OnroadBlocksPool) addSimpleCache(addr types.Address, accountInfo *OnroadAccountInfo) {
	p.simpleCache.Store(addr, accountInfo)

	timer, ok := p.simpleCacheDeadTimer.Load(addr)
	if ok && timer != nil {
		p.log.Info("addSimpleCache Reset timer")
		timer.(*time.Timer).Reset(simpleCacheExpireTime)
	} else {
		p.simpleCacheDeadTimer.Store(addr, time.AfterFunc(simpleCacheExpireTime, func() {
			p.log.Info("simple cache end life delete it", "addr", addr)
			p.simpleCache.Delete(addr)
		}))
	}
}

func (p *OnroadBlocksPool) GetOnroadAccountInfo(addr types.Address) (*OnroadAccountInfo, error) {
	p.log.Debug("first load in simple cache", "addr", addr)
	if c, ok := p.simpleCache.Load(addr); ok {
		v, ok := p.simpleCacheDeadTimer.Load(addr)
		if ok {
			v.(*time.Timer).Reset(simpleCacheExpireTime)
		}
		return c.(*OnroadAccountInfo), nil
	}

	p.log.Debug("second load from full cache", "addr", addr)
	if fullcache, ok := p.fullCache.Load(addr); ok {
		accountInfo := fullcache.(*onroadBlocksCache).toOnroadAccountInfo(addr)
		if accountInfo != nil {
			p.addSimpleCache(addr, accountInfo)
			return accountInfo, nil
		}
	}

	p.log.Debug("third load from db", "addr", addr)
	accountInfo, e := p.dbAccess.GetCommonAccInfo(&addr)
	if e != nil {
		return nil, e
	}
	if accountInfo != nil {
		p.addSimpleCache(addr, accountInfo)
	}

	return accountInfo, nil

}

func (p *OnroadBlocksPool) GetNextCommonTx(addr types.Address) *ledger.AccountBlock {
	p.log.Debug("GetNextCommonTx", "addr", addr)
	c, ok := p.fullCache.Load(addr)
	if !ok {
		return nil
	}
	return c.(*onroadBlocksCache).GetNextTx()
}

func (p *OnroadBlocksPool) ResetCacheCursor(addr types.Address) {
	p.log.Debug("ResetCacheCursor", "addr", addr)
	c, ok := p.fullCache.Load(addr)
	if !ok {
		return
	}
	c.(*onroadBlocksCache).ResetCursor()
}

func (p *OnroadBlocksPool) loadFullCacheFromDb(addr types.Address) error {
	blocks, e := p.dbAccess.GetAllOnroadBlocks(addr)
	if e != nil {
		return e
	}
	p.log.Debug("get from db", "len", len(blocks))

	list := list.New()
	for _, value := range blocks {
		if value != nil {
			list.PushBack(value)
		}
	}

	cache := &onroadBlocksCache{
		blocks:         *list,
		currentEle:     list.Front(),
		referenceCount: 1,
	}
	p.fullCache.Store(addr, cache)

	return nil
}

func (p *OnroadBlocksPool) AcquireFullOnroadBlocksCache(addr types.Address) {
	log := p.log.New("AcquireFullOnroadBlocksCache", addr)
	if t, ok := p.fullCacheDeadTimer.Load(addr); ok {
		if t != nil {
			log.Debug("stop timer")
			t.(*time.Timer).Stop()
		}
	}
	// first load in cache
	if c, ok := p.fullCache.Load(addr); ok {
		c.(*onroadBlocksCache).addReferenceCount()
		log.Debug("found in cache", "ref", c.(*onroadBlocksCache).referenceCount)
		return
	}

	// second load in db
	if e := p.loadFullCacheFromDb(addr); e != nil {
		log.Error(e.Error())
	}
}

func (p *OnroadBlocksPool) ReleaseFullOnroadBlocksCache(addr types.Address) error {
	log := p.log.New("ReleaseFullOnroadBlocksCache", addr)
	v, ok := p.fullCache.Load(addr)
	if !ok {
		log.Debug("no cache found")
		return nil
	}
	c := v.(*onroadBlocksCache)
	if c.subReferenceCount() <= 0 {
		log.Debug("cache found ref <= 0 delete cache")

		c.ResetCursor()
		p.fullCacheDeadTimer.Store(addr, time.AfterFunc(fullCacheExpireTime, func() {
			log.Debug("cache delete")
			p.fullCache.Delete(addr)
		}))
		return nil
	}
	log.Debug("after release", "ref", c.referenceCount)

	return nil
}

func (p *OnroadBlocksPool) WriteOnroadSuccess(blocks []*vm_context.VmAccountBlock) {
	for _, v := range blocks {
		if v.AccountBlock.IsSendBlock() {
			p.updateCache(true, v.AccountBlock)
		} else {
			p.updateCache(false, v.AccountBlock)
		}
	}
}

func (p *OnroadBlocksPool) WriteOnroad(batch *leveldb.Batch, blockList []*vm_context.VmAccountBlock) error {
	for _, v := range blockList {
		if v.AccountBlock.IsSendBlock() {
			// basic writeMeta func
			if err := p.dbAccess.writeOnroadMeta(batch, v.AccountBlock); err != nil {
				p.log.Error("writeOnroadMeta", "error", err)
				return err
			}

			if v.AccountBlock.BlockType == ledger.BlockTypeSendCreate {
				unsavedCache := v.VmContext.UnsavedCache()
				gidList := unsavedCache.ContractGidList()
				for _, v := range gidList {
					if err := p.dbAccess.WriteContractAddrToGid(batch, *v.Gid(), *v.Addr()); err != nil {
						p.log.Error("WriteContractAddrToGid", "error", err)
						return err
					}
				}
			}
		} else {
			if err := p.dbAccess.deleteOnroadMeta(batch, v.AccountBlock); err != nil {
				p.log.Error("deleteOnroadMeta", "error", err)
				return err
			}
		}
	}

	return nil
}

func (p *OnroadBlocksPool) DeleteDirect(sendBlock *ledger.AccountBlock) error {
	return p.dbAccess.store.DeleteMeta(nil, &sendBlock.ToAddress, &sendBlock.Hash)
}


func (p *OnroadBlocksPool) RevertOnroadSuccess(subLedger map[types.Address][]*ledger.AccountBlock) {
	//async update the cache
	go func() {
		for addr, _ := range subLedger {
			// if the full cache is in hold by a worker we will rebuild it else we delete it
			if cache, ok := p.fullCache.Load(addr); ok {
				c := cache.(*onroadBlocksCache)
				if c.referenceCount > 0 {
					p.loadFullCacheFromDb(addr)
				} else {
					if t, ok := p.fullCacheDeadTimer.Load(addr); ok {
						t.(*time.Timer).Stop()
						p.fullCacheDeadTimer.Delete(addr)
					}
					p.fullCache.Delete(addr)
				}
			}

			p.deleteSimpleCache(addr)
		}

	}()
}

// RevertOnroad means to revert according to bifurcation
func (p *OnroadBlocksPool) RevertOnroad(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error {

	cutMap := excludeSubordinate(subLedger)
	for _, blocks := range cutMap {
		// the blockList is sorted by height with ascending order
		for i := len(blocks); i > 0; i-- {
			v := blocks[i]

			if v.IsReceiveBlock() {
				sendBlock, err := p.dbAccess.Chain.GetAccountBlockByHash(&v.FromBlockHash)
				if err != nil {
					p.log.Error("GetAccountBlockByHash", "error", err)
					return err
				}
				if err := p.dbAccess.writeOnroadMeta(batch, sendBlock); err != nil {
					p.log.Error("revert receiveBlock failed", "error", err)
					return err
				}
			} else {
				if err := p.dbAccess.deleteOnroadMeta(batch, v); err != nil {
					p.log.Error("revert the sendBlock's and the referred failed", "error", err)
					return err
				}

				gid := contracts.GetGidFromCreateContractData(v.Data)
				p.dbAccess.DeleteContractAddrFromGid(batch, gid, v.ToAddress)
			}
		}
	}
	return nil
}

func excludeSubordinate(subLedger map[types.Address][]*ledger.AccountBlock) map[types.Hash][]*ledger.AccountBlock {
	var cutMap map[types.Hash][]*ledger.AccountBlock
	for _, blockList := range subLedger {
		for _, v := range blockList {
			if v.IsSendBlock() {
				if _, ok := cutMap[v.Hash]; ok {
					cutMap[v.Hash] = nil
				}
				cutMap[v.Hash] = append(cutMap[v.Hash], v)
			} else {
				if bl, ok := cutMap[v.FromBlockHash]; ok && bl[len(bl)].IsSendBlock() {
					continue
				}
				cutMap[v.FromBlockHash] = append(cutMap[v.FromBlockHash], v)
			}
		}
	}
	return cutMap
}

func (p *OnroadBlocksPool) updateFullCache(writeType bool, block *ledger.AccountBlock) {
	if v, ok := p.fullCache.Load(block.AccountAddress); ok {
		fullCache := v.(*onroadBlocksCache)
		if writeType {
			fullCache.addTx(block)
		} else {
			fullCache.rmTx(block)
		}
	}
}

func (p *OnroadBlocksPool) updateSimpleCache(writeType bool, block *ledger.AccountBlock) {

	value, ok := p.simpleCache.Load(block.AccountAddress)
	if !ok {
		return
	}

	simpleAccountInfo := value.(*OnroadAccountInfo)
	simpleAccountInfo.mutex.Lock()
	defer simpleAccountInfo.mutex.Unlock()

	tokenBalanceInfo, ok := simpleAccountInfo.TokenBalanceInfoMap[block.TokenId]
	if writeType {
		if ok {
			tokenBalanceInfo.TotalAmount.Add(&tokenBalanceInfo.TotalAmount, block.Amount)
			tokenBalanceInfo.Number += 1
		} else {
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId].TotalAmount = *block.Amount
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId].Number = 1
		}
		simpleAccountInfo.TotalNumber += 1
	} else {
		if ok {
			if tokenBalanceInfo.TotalAmount.Cmp(block.Amount) == -1 {
				p.log.Error("conflict with the memory info, so can't update when writeType is false")
				p.deleteSimpleCache(block.AccountAddress)
			}
			if tokenBalanceInfo.TotalAmount.Cmp(block.Amount) == 0 {
				delete(simpleAccountInfo.TokenBalanceInfoMap, block.TokenId)
			} else {
				tokenBalanceInfo.TotalAmount.Sub(&tokenBalanceInfo.TotalAmount, block.Amount)
			}
		}
		simpleAccountInfo.TotalNumber -= 1
		tokenBalanceInfo.Number -= 1
	}
}

func (p *OnroadBlocksPool) deleteSimpleCache(addr types.Address) {
	if t, ok := p.simpleCacheDeadTimer.Load(addr); ok {
		t.(*time.Timer).Stop()
		p.simpleCacheDeadTimer.Delete(addr)

	}
	if _, ok := p.simpleCache.Load(addr); ok {
		p.simpleCache.Delete(addr)
	}
}

func (p *OnroadBlocksPool) updateCache(writeType bool, block *ledger.AccountBlock) {

	p.updateFullCache(writeType, block)

	p.updateSimpleCache(writeType, block)

}

func (p *OnroadBlocksPool) NewSignalToWorker(block *ledger.AccountBlock) {
	gid, err := p.dbAccess.Chain.GetContractGid(&block.AccountAddress)
	if err != nil {
		p.log.Error("NewSignalToWorker", "err", err)
		return
	}
	if gid != nil {
		p.contractListenerMutex.RLock()
		defer p.contractListenerMutex.RUnlock()
		if f, ok := p.newContractListener[*gid]; ok {
			f(block.AccountAddress)
		}
	} else {
		p.commonTxListenerMutex.RLock()
		defer p.commonTxListenerMutex.RUnlock()
		if f, ok := p.newCommonTxListener[block.AccountAddress]; ok {
			f()
		}
	}
}

func (p *OnroadBlocksPool) Close() error {
	p.log.Info("Close()")

	p.simpleCacheDeadTimer.Range(func(_, value interface{}) bool {
		if value != nil {
			value.(*time.Timer).Stop()
		}
		return true
	})
	p.simpleCache = nil

	p.fullCacheDeadTimer.Range(func(_, value interface{}) bool {
		if value != nil {
			value.(*time.Timer).Stop()
		}
		return true
	})
	p.fullCache = nil

	p.log.Info("Close() end")
	return nil
}

func (p *OnroadBlocksPool) AddCommonTxLis(addr types.Address, f func()) {
	p.commonTxListenerMutex.Lock()
	defer p.commonTxListenerMutex.Unlock()
	p.newCommonTxListener[addr] = f
}

func (p *OnroadBlocksPool) RemoveCommonTxLis(addr types.Address) {
	p.commonTxListenerMutex.Lock()
	defer p.commonTxListenerMutex.Unlock()
	delete(p.newCommonTxListener, addr)
}

func (p *OnroadBlocksPool) AddContractLis(gid types.Gid, f func(address types.Address)) {
	p.contractListenerMutex.Lock()
	defer p.contractListenerMutex.Unlock()
	p.newContractListener[gid] = f
}

func (p *OnroadBlocksPool) RemoveContractLis(gid types.Gid) {
	p.contractListenerMutex.Lock()
	defer p.contractListenerMutex.Unlock()
	delete(p.newContractListener, gid)
}
