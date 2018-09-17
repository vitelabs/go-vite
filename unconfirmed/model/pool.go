package model

import (
	"container/list"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

const (
	fullCacheExpireTime   = 2 * time.Minute
	simpleCacheExpireTime = 20 * time.Minute
)

// obtaining the account info from cache or db and manage the cache lifecycle
type UnconfirmedBlocksPool struct {
	dbAccess *UAccess

	fullCache          map[types.Address]*unconfirmedBlocksCache
	fullCacheDeadTimer map[types.Address]*time.Timer
	fullCacheMutex     sync.RWMutex

	simpleCache          map[types.Address]*CommonAccountInfo
	simpleCacheDeadTimer map[types.Address]*time.Timer
	simpleCacheMutex     sync.RWMutex

	newCommonTxListener map[types.Address]func()
	newContractListener map[types.Gid]func()

	log log15.Logger
}

func NewUnconfirmedBlocksPool(dbAccess *UAccess) *UnconfirmedBlocksPool {
	return &UnconfirmedBlocksPool{
		dbAccess:             dbAccess,
		fullCache:            make(map[types.Address]*unconfirmedBlocksCache),
		fullCacheDeadTimer:   make(map[types.Address]*time.Timer),
		simpleCache:          make(map[types.Address]*CommonAccountInfo),
		simpleCacheDeadTimer: make(map[types.Address]*time.Timer),
		newCommonTxListener:  make(map[types.Address]func()),
		newContractListener:  make(map[types.Gid]func()),
		log:                  log15.New("unconfirmed", "UnconfirmedBlocksPool"),
	}
}

func (p *UnconfirmedBlocksPool) GetAddrListByGid(gid types.Gid) (addrList []*types.Address, err error) {
	return p.dbAccess.GetAddrListByGid(gid)
}

func (p *UnconfirmedBlocksPool) Close() error {
	p.log.Info("Close()")
	p.simpleCacheMutex.Lock()
	for _, v := range p.simpleCacheDeadTimer {
		if v != nil {
			v.Stop()
		}
	}
	p.simpleCache = nil
	p.simpleCacheMutex.Unlock()

	p.fullCacheMutex.Lock()
	for _, v := range p.fullCacheDeadTimer {
		if v != nil {
			v.Stop()
		}
	}
	p.fullCache = nil
	p.fullCacheMutex.Unlock()
	p.log.Info("Close() end")
	return nil
}

func (p *UnconfirmedBlocksPool) addSimpleCache(addr types.Address, accountInfo *CommonAccountInfo) {
	p.log.Info("addSimpleCache", "addr", addr, "TotalNumber", accountInfo.TotalNumber)
	p.simpleCacheMutex.Lock()
	p.simpleCache[addr] = accountInfo
	p.simpleCacheMutex.Unlock()

	timer, ok := p.simpleCacheDeadTimer[addr]
	if ok && timer != nil {
		p.log.Info("addSimpleCache Reset timer")
		timer.Reset(simpleCacheExpireTime)
	} else {
		p.simpleCacheDeadTimer[addr] = time.AfterFunc(simpleCacheExpireTime, func() {
			p.log.Info("simple cache end life delete it", "addr", addr)
			p.simpleCacheMutex.Lock()
			delete(p.simpleCache, addr)
			p.simpleCacheMutex.Unlock()
		})
	}
}

func (p *UnconfirmedBlocksPool) GetCommonAccountInfo(addr types.Address) (*CommonAccountInfo, error) {
	p.log.Info("first load in simple cache", "addr", addr)
	p.simpleCacheMutex.RLock()
	if c, ok := p.simpleCache[addr]; ok {
		p.simpleCacheDeadTimer[addr].Reset(simpleCacheExpireTime)
		p.simpleCacheMutex.RUnlock()
		return c, nil
	}
	p.simpleCacheMutex.RUnlock()

	p.log.Info("second load from full cache", "addr", addr)
	p.fullCacheMutex.RLock()
	defer p.fullCacheMutex.RUnlock()
	if fullcache, ok := p.fullCache[addr]; ok {
		accountInfo := fullcache.toCommonAccountInfo(p.dbAccess.Chain.GetTokenInfoById)
		if accountInfo != nil {
			p.addSimpleCache(addr, accountInfo)
			return accountInfo, nil
		}
	}

	p.log.Info("third load from db", "addr", addr)
	accountInfo, e := p.dbAccess.GetCommonAccInfo(&addr)
	if e != nil {
		return nil, e
	}
	if accountInfo != nil {
		p.addSimpleCache(addr, accountInfo)
	}

	return accountInfo, nil

}

func (p *UnconfirmedBlocksPool) GetNextTx(addr types.Address) *ledger.AccountBlock {
	p.log.Info("GetNextTx", "addr", addr)
	p.fullCacheMutex.RLock()
	defer p.fullCacheMutex.RUnlock()
	c, ok := p.fullCache[addr]
	if !ok {
		p.fullCacheMutex.RUnlock()
		return nil
	}
	return c.GetNextTx()
}

func (p *UnconfirmedBlocksPool) ResetCacheCursor(addr types.Address) {
	p.log.Info("ResetCacheCursor", "addr", addr)
	p.fullCacheMutex.RLock()
	defer p.fullCacheMutex.RUnlock()
	c, ok := p.fullCache[addr]
	if !ok {
		p.fullCacheMutex.RUnlock()
		return
	}
	c.ResetCursor()
}

func (p *UnconfirmedBlocksPool) AcquireAccountInfoCache(addr types.Address) error {
	log := p.log.New("AcquireAccountInfoCache", addr)
	p.fullCacheMutex.RLock()

	if t, ok := p.fullCacheDeadTimer[addr]; ok {
		if t != nil {
			log.Info("stop timer")
			t.Stop()
		}
	}

	if c, ok := p.fullCache[addr]; ok {
		c.addReferenceCount()
		log.Info("found in cache", "ref", c.referenceCount)
		p.fullCacheMutex.RUnlock()
		return nil
	}
	p.fullCacheMutex.RUnlock()

	p.fullCacheMutex.Lock()
	defer p.fullCacheMutex.Unlock()
	blocks, e := p.dbAccess.GetAllUnconfirmedBlocks(addr)
	if e != nil {
		log.Error("get from db", "err", e)
		return e
	}
	log.Info("get from db", "len", len(blocks))

	list := list.New()
	for _, value := range blocks {
		list.PushBack(value)
	}

	p.fullCache[addr] = &unconfirmedBlocksCache{
		blocks:         *list,
		currentEle:     list.Front(),
		referenceCount: 1,
	}

	return nil
}

func (p *UnconfirmedBlocksPool) ReleaseAccountInfoCache(addr types.Address) error {
	log := p.log.New("ReleaseAccountInfoCache", addr)
	p.fullCacheMutex.RLock()
	c, ok := p.fullCache[addr]
	if !ok {
		log.Info("no cache found")
		p.fullCacheMutex.RUnlock()
		return nil
	}
	if c.subReferenceCount() <= 0 {
		log.Info("cache found ref <= 0 delete cache")
		p.fullCacheMutex.RUnlock()

		p.fullCache[addr].ResetCursor()
		p.fullCacheDeadTimer[addr] = time.AfterFunc(fullCacheExpireTime, func() {
			log.Info("cache delete")
			p.DeleteFullCache(addr)
		})
		return nil
	}
	log.Info("after release", "ref", c.referenceCount)
	p.fullCacheMutex.RUnlock()

	return nil
}

func (p *UnconfirmedBlocksPool) DeleteFullCache(address types.Address) {
	p.fullCacheMutex.Lock()
	defer p.fullCacheMutex.Unlock()
	delete(p.fullCache, address)
}

func (p *UnconfirmedBlocksPool) WriteUnconfirmed(writeType bool, batch *leveldb.Batch, block *ledger.AccountBlock) error {
	p.log.Info("WriteUnconfirmed ", "writeType", writeType)
	if writeType {
		if err := p.dbAccess.writeUnconfirmedMeta(batch, block); err != nil {
			p.log.Error("writeUnconfirmedMeta", "error", err)
			return err
		}

		// fixme: @gx whether need to wait the block insert into Chain and try the following
		p.NewSignalToWorker(block)
	} else { // delete
		if err := p.dbAccess.deleteUnconfirmedMeta(batch, block); err != nil {
			p.log.Error("deleteUnconfirmedMeta", "error", err)
			return err
		}
	}

	// fixme: @gx whether need to wait the block insert into Chain and try the following
	p.updateCache(writeType, block)

	return nil
}

func (p *UnconfirmedBlocksPool) updateFullCache(writeType bool, block *ledger.AccountBlock) error {
	p.fullCacheMutex.Lock()
	defer p.fullCacheMutex.Unlock()

	fullCache, ok := p.fullCache[block.ToAddress]
	if !ok || fullCache.blocks.Len() == 0 {
		p.log.Info("updateCache：no fullCache")
		return nil
	}

	if writeType {
		fullCache.addTx(block)
	} else {
		fullCache.rmTx(block)
	}

	return nil
}

func (p *UnconfirmedBlocksPool) updateSimpleCache(writeType bool, block *ledger.AccountBlock) error {
	p.simpleCacheMutex.Lock()
	defer p.simpleCacheMutex.Unlock()

	simpleAccountInfo, ok := p.simpleCache[block.ToAddress]
	if !ok {
		p.log.Info("updateSimpleCache：no cache")
		return nil
	}

	tokenBalanceInfo, ok := simpleAccountInfo.TokenBalanceInfoMap[block.TokenId]
	if writeType {
		if ok {
			tokenBalanceInfo.TotalAmount.Add(&tokenBalanceInfo.TotalAmount, block.Amount)
			tokenBalanceInfo.Number += 1
		} else {
			token, err := p.dbAccess.Chain.GetTokenInfoById(&block.TokenId)
			if err != nil {
				return errors.New("func UpdateCommonAccInfo.GetByTokenId failed" + err.Error())
			}
			if token == nil {
				return errors.New("func UpdateCommonAccInfo.GetByTokenId failed token nil")
			}
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId].Token = *token
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId].TotalAmount = *block.Amount
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId].Number = 1
		}
		simpleAccountInfo.TotalNumber += 1
	} else {
		if ok {
			if tokenBalanceInfo.TotalAmount.Cmp(block.Amount) == -1 {
				return errors.New("conflict with the memory info, so can't update when writeType is false")
			}
			if tokenBalanceInfo.TotalAmount.Cmp(block.Amount) == 0 {
				delete(simpleAccountInfo.TokenBalanceInfoMap, block.TokenId)
			} else {
				tokenBalanceInfo.TotalAmount.Sub(&tokenBalanceInfo.TotalAmount, block.Amount)
			}
		} else {
			p.log.Info("find no memory tokenInfo, so can't update when writeType is false")
		}
		simpleAccountInfo.TotalNumber -= 1
		tokenBalanceInfo.Number -= 1
	}

	return nil
}

func (p *UnconfirmedBlocksPool) updateCache(writeType bool, block *ledger.AccountBlock) {
	e := p.updateFullCache(writeType, block)
	if e != nil {
		p.log.Error("updateFullCache", "err", e)
	}

	e = p.updateSimpleCache(writeType, block)
	if e != nil {
		p.log.Error("updateSimpleCache", "err", e)
	}
}

func (p *UnconfirmedBlocksPool) NewSignalToWorker(block *ledger.AccountBlock) {
	// todo @lyd will support it
	gid := p.dbAccess.Chain.GetGid(block.AccountAddress)
	if gid != nil {
		if f, ok := p.newContractListener[gid]; ok {
			f()
		}
	} else {
		if f, ok := p.newCommonTxListener[block.ToAddress]; ok {
			f()
		}
	}
}

func (p *UnconfirmedBlocksPool) GetUnconfirmedBlocks(index, num, count uint64, addr *types.Address) (blockList []*ledger.AccountBlock, err error) {
	return p.dbAccess.GetUnconfirmedBlocks(index, num, count, addr)
}

func (p *UnconfirmedBlocksPool) AddCommonTxLis(addr types.Address, f func()) {
	p.newCommonTxListener[addr] = f
}

func (p *UnconfirmedBlocksPool) RemoveCommonTxLis(addr types.Address) {
	delete(p.newCommonTxListener, addr)
}

func (p *UnconfirmedBlocksPool) AddContractLis(gid types.Gid, f func()) {
	p.newContractListener[gid] = f
}

func (p *UnconfirmedBlocksPool) RemoveContractLis(gid types.Gid) {
	delete(p.newContractListener, gid)
}
