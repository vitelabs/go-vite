package model

import (
	"container/list"
	"fmt"
	"github.com/vitelabs/go-vite/vm/util"
	"sync"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_context"
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

	contractCache *sync.Map //map[types.Address]*ContractCallerList

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
		contractCache:        &sync.Map{},
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
		blocks:         list,
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
		log.Debug("found in cache", "ref", c.(*onroadBlocksCache).getReferenceCount())
		return
	}

	// second load in db
	if e := p.loadFullCacheFromDb(addr); e != nil {
		log.Error(e.Error())
	}
}

func (p *OnroadBlocksPool) GetNextCommonTx(addr types.Address) *ledger.AccountBlock {
	p.log.Debug("GetNextCommonTx", "addr", addr)
	c, ok := p.fullCache.Load(addr)
	if !ok {
		return nil
	}
	return c.(*onroadBlocksCache).GetNextTx()
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
	log.Debug("after release", "ref", c.getReferenceCount())

	return nil
}

func (p *OnroadBlocksPool) loadContractCacheFromDb(addr types.Address) error {
	p.log.Debug("loadContractCacheFromDb", "addr", addr)
	if c, ok := p.contractCache.Load(addr); ok {
		if cc, ok := c.(*ContractCallerList); ok && cc != nil {
			p.log.Debug(fmt.Sprintf("found in cache, tx remain=%v", cc.TxRemain()))
			return nil
		}
		// refactoring cache
		p.contractCache.Delete(addr)
	}

	blockList, e := p.dbAccess.GetAllOnroadBlocks(addr)
	if e != nil {
		return e
	}
	p.log.Debug(fmt.Sprintf("get contract onroad %v from db, len=%v", addr, len(blockList)))
	if len(blockList) <= 0 {
		return nil
	}
	contractBlocksList := &ContractCallerList{list: make([]*contractCallerBlocks, 0)}
	for k, b := range blockList {
		p.log.Debug(fmt.Sprintf("add block[%v]: addr=%v height=%v blockHash=%v ", k, b.AccountAddress, b.Height, b.Hash))
		contractBlocksList.AddNewTx(b)
	}
	p.contractCache.Store(addr, contractBlocksList)
	p.log.Debug(fmt.Sprintf("load result: len=%v", contractBlocksList.TxRemain()))
	return nil
}

func (p *OnroadBlocksPool) AcquireOnroadSortedContractCache(addr types.Address) {
	log := p.log.New("AcquireOnroadSortedContractCache", addr)
	if e := p.loadContractCacheFromDb(addr); e != nil {
		log.Error(e.Error())
	}
}

func (p *OnroadBlocksPool) GetNextContractTx(addr types.Address) *ledger.AccountBlock {
	p.log.Debug("GetNextContractTx", "addr", addr)
	if c, ok := p.contractCache.Load(addr); ok {
		if cc, ok := c.(*ContractCallerList); ok && cc != nil {
			b := cc.GetNextTx()
			p.log.Debug(fmt.Sprintf("currentCallerIndex=%v", cc.GetCurrentIndex()))
			return b
		}
	}
	return nil
}

func (p *OnroadBlocksPool) GetContractCallerList(addr types.Address) *ContractCallerList {
	if c, ok := p.contractCache.Load(addr); ok {
		if cc, ok := c.(*ContractCallerList); ok && cc != nil {
			return cc
		}
	}
	return nil
}

func (p *OnroadBlocksPool) ReleaseContractCache(addr types.Address) {
	log := p.log.New("ReleaseContractCache", addr)
	if v, ok := p.contractCache.Load(addr); ok {
		if l, ok := v.(*ContractCallerList); ok && l != nil {
			log.Debug(fmt.Sprintf("release %v callers", l.Len()), "tx num remain", l.TxRemain())
		}
		p.contractCache.Delete(addr)
	}
}

func (p *OnroadBlocksPool) DeleteContractCache(gid types.Gid) {
	p.log.Debug("DeleteContractCache", "gid", gid)
	if p.contractCache != nil {
		p.contractCache = &sync.Map{}
	}
}

func (p *OnroadBlocksPool) WriteOnroadSuccess(blocks []*vm_context.VmAccountBlock) {
	for _, v := range blocks {
		if v.AccountBlock.IsSendBlock() {
			code, _ := p.dbAccess.Chain.AccountType(&v.AccountBlock.ToAddress)
			if (code == ledger.AccountTypeNotExist && v.AccountBlock.BlockType == ledger.BlockTypeSendCreate) ||
				code == ledger.AccountTypeContract || code == ledger.AccountTypeError {
				return
			}
			p.updateCache(true, v.AccountBlock)
			p.NewSignalToWorker(v.AccountBlock)
		} else {
			code, _ := p.dbAccess.Chain.AccountType(&v.AccountBlock.AccountAddress)
			if code == ledger.AccountTypeGeneral {
				p.updateCache(false, v.AccountBlock)
			}
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
					p.log.Debug("WriteOnroad", "gid", v.Gid(), "addr", v.Addr())
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
	cutMap := excludeSubordinate(subLedger)
	for _, blocks := range cutMap {
		for i := len(blocks) - 1; i >= 0; i-- {
			v := blocks[i]
			addr := types.Address{}
			if v.IsReceiveBlock() {
				addr = v.AccountAddress
			} else {
				addr = v.ToAddress
			}
			p.deleteSimpleCache(addr)
			p.deleteFullCache(addr)
		}
	}
}

// RevertOnroad means to revert according to bifurcation
func (p *OnroadBlocksPool) RevertOnroad(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error {

	cutMap := excludeSubordinate(subLedger)
	for _, blocks := range cutMap {
		// the blockList is sorted by height with ascending order
		for i := len(blocks) - 1; i >= 0; i-- {
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
				if v.BlockType == ledger.BlockTypeSendCreate {
					gid := util.GetGidFromCreateContractData(v.Data)
					p.dbAccess.DeleteContractAddrFromGid(batch, gid, v.ToAddress)
				}
			}
		}
	}
	return nil
}

func excludeSubordinate(subLedger map[types.Address][]*ledger.AccountBlock) map[types.Hash][]*ledger.AccountBlock {
	cutMap := make(map[types.Hash][]*ledger.AccountBlock)
	for _, blockList := range subLedger {
		for _, v := range blockList {
			if v.IsSendBlock() {
				if _, ok := cutMap[v.Hash]; ok {
					cutMap[v.Hash] = nil
				}
				cutMap[v.Hash] = append(cutMap[v.Hash], v)
			} else {
				if bl, _ := cutMap[v.FromBlockHash]; len(bl) > 0 && bl[len(bl)-1].IsSendBlock() {
					continue
				}
				cutMap[v.FromBlockHash] = append(cutMap[v.FromBlockHash], v)
			}
		}
	}
	return cutMap
}

func (p *OnroadBlocksPool) updateFullCache(writeType bool, block *ledger.AccountBlock) {
	if writeType {
		if v, ok := p.fullCache.Load(block.ToAddress); ok {
			fullCache := v.(*onroadBlocksCache)
			fullCache.addTx(block)
		}
	} else {
		if v, ok := p.fullCache.Load(block.AccountAddress); ok {
			fullCache := v.(*onroadBlocksCache)
			fullCache.rmTx(block)
		}
	}
}

func (p *OnroadBlocksPool) updateSimpleCache(isAdd bool, block *ledger.AccountBlock) {

	if isAdd {
		value, ok := p.simpleCache.Load(block.ToAddress)
		if !ok {
			return
		}
		simpleAccountInfo := value.(*OnroadAccountInfo)
		simpleAccountInfo.mutex.Lock()
		defer simpleAccountInfo.mutex.Unlock()

		tokenBalanceInfo, ok := simpleAccountInfo.TokenBalanceInfoMap[block.TokenId]
		if ok {
			tokenBalanceInfo.TotalAmount.Add(&tokenBalanceInfo.TotalAmount, block.Amount)
			tokenBalanceInfo.Number += 1
		} else {
			var tinfo TokenBalanceInfo
			tinfo.TotalAmount = *block.Amount
			tinfo.Number = 1
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId] = &tinfo
		}
		simpleAccountInfo.TotalNumber += 1
	} else {
		value, ok := p.simpleCache.Load(block.AccountAddress)
		if !ok {
			return
		}
		simpleAccountInfo := value.(*OnroadAccountInfo)
		simpleAccountInfo.mutex.Lock()
		defer simpleAccountInfo.mutex.Unlock()

		accountBlock, e := p.dbAccess.Chain.GetAccountBlockByHash(&block.FromBlockHash)
		if e != nil {
			p.log.Error("updateSimpleCache GetAccountBlockByHash ", "err", e)
			p.deleteSimpleCache(block.AccountAddress)
			return
		}

		if accountBlock == nil {
			p.log.Error("updateSimpleCache GetAccountBlockByHash get an empty block")
			p.deleteSimpleCache(block.AccountAddress)
			return
		}
		amount := accountBlock.Amount
		tti := accountBlock.TokenId

		tokenBalanceInfo, ok := simpleAccountInfo.TokenBalanceInfoMap[tti]
		if ok {
			if tokenBalanceInfo.TotalAmount.Cmp(amount) == -1 {
				p.log.Error("conflict with the memory info, so can't update when isAdd is false")
				p.deleteSimpleCache(block.AccountAddress)
				return
			}
			if tokenBalanceInfo.TotalAmount.Cmp(amount) == 0 {
				delete(simpleAccountInfo.TokenBalanceInfoMap, tti)
			} else {
				tokenBalanceInfo.TotalAmount.Sub(&tokenBalanceInfo.TotalAmount, amount)
			}
			tokenBalanceInfo.Number -= 1
			simpleAccountInfo.TotalNumber -= 1
		}
	}
}

func (p *OnroadBlocksPool) deleteSimpleCache(addr types.Address) {
	if p.simpleCacheDeadTimer != nil {
		if t, ok := p.simpleCacheDeadTimer.Load(addr); ok {
			if t != nil {
				t.(*time.Timer).Stop()
			}
			p.simpleCacheDeadTimer.Delete(addr)

		}
	}
	if p.simpleCache != nil {
		if _, ok := p.simpleCache.Load(addr); ok {
			p.simpleCache.Delete(addr)
		}
	}
}

func (p *OnroadBlocksPool) deleteFullCache(addr types.Address) {
	if p.fullCacheDeadTimer != nil {
		if t, ok := p.fullCacheDeadTimer.Load(addr); ok {
			if t != nil {
				t.(*time.Timer).Stop()
			}
			p.fullCache.Delete(addr)
		}
	}
	if p.fullCache != nil {
		if _, ok := p.fullCache.Load(addr); ok {
			p.fullCache.Delete(addr)
		}
	}
}

func (p *OnroadBlocksPool) updateCache(writeType bool, block *ledger.AccountBlock) {

	p.updateFullCache(writeType, block)

	p.updateSimpleCache(writeType, block)

}

func (p *OnroadBlocksPool) NewSignalToWorker(block *ledger.AccountBlock) {
	gid, err := p.dbAccess.Chain.GetContractGid(&block.ToAddress)
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
		if f, ok := p.newCommonTxListener[block.ToAddress]; ok {
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
	p.contractCache = nil

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
