package model

import (
	"container/list"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
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

	fullCache          *sync.Map // map[types.Address]*unconfirmedBlocksCache
	fullCacheDeadTimer *sync.Map // map[types.Address]*time.Timer

	simpleCache          *sync.Map // map[types.Address]*CommonAccountInfo
	simpleCacheDeadTimer *sync.Map //map[types.Address]*time.Timer

	newCommonTxListener   map[types.Address]func()
	commonTxListenerMutex sync.RWMutex

	newContractListener   map[types.Gid]func()
	contractListenerMutex sync.RWMutex

	log log15.Logger
}

func NewUnconfirmedBlocksPool(dbAccess *UAccess) *UnconfirmedBlocksPool {
	return &UnconfirmedBlocksPool{
		dbAccess:             dbAccess,
		fullCache:            &sync.Map{},
		fullCacheDeadTimer:   &sync.Map{},
		simpleCache:          &sync.Map{},
		simpleCacheDeadTimer: &sync.Map{},
		newCommonTxListener:  make(map[types.Address]func()),
		newContractListener:  make(map[types.Gid]func()),
		log:                  log15.New("unconfirmed", "UnconfirmedBlocksPool"),
	}
}

func (p *UnconfirmedBlocksPool) GetAddrListByGid(gid types.Gid) (addrList []*types.Address, err error) {
	return p.dbAccess.GetContractAddrListByGid(&gid)
}

func (p *UnconfirmedBlocksPool) Close() error {
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

func (p *UnconfirmedBlocksPool) addSimpleCache(addr types.Address, accountInfo *CommonAccountInfo) {
	//p.log.Info("addSimpleCache", "addr", addr, "TotalNumber", accountInfo.TotalNumber)
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

func (p *UnconfirmedBlocksPool) GetCommonAccountInfo(addr types.Address) (*CommonAccountInfo, error) {
	p.log.Info("first load in simple cache", "addr", addr)
	if c, ok := p.simpleCache.Load(addr); ok {
		v, ok := p.simpleCacheDeadTimer.Load(addr)
		if ok {
			v.(*time.Timer).Reset(simpleCacheExpireTime)
		}
		return c.(*CommonAccountInfo), nil
	}

	p.log.Info("second load from full cache", "addr", addr)
	if fullcache, ok := p.fullCache.Load(addr); ok {
		accountInfo := fullcache.(*unconfirmedBlocksCache).toCommonAccountInfo(p.dbAccess.Chain.GetTokenInfoById)
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
	c, ok := p.fullCache.Load(addr)
	if !ok {
		return nil
	}
	return c.(*unconfirmedBlocksCache).GetNextTx()
}

func (p *UnconfirmedBlocksPool) ResetCacheCursor(addr types.Address) {
	p.log.Info("ResetCacheCursor", "addr", addr)
	c, ok := p.fullCache.Load(addr)
	if !ok {
		return
	}
	c.(*unconfirmedBlocksCache).ResetCursor()
}

func (p *UnconfirmedBlocksPool) AcquireAccountInfoCache(addr types.Address) error {
	log := p.log.New("AcquireAccountInfoCache", addr)
	if t, ok := p.fullCacheDeadTimer.Load(addr); ok {
		if t != nil {
			log.Info("stop timer")
			t.(*time.Timer).Stop()
		}
	}

	if c, ok := p.fullCache.Load(addr); ok {
		c.(*unconfirmedBlocksCache).addReferenceCount()
		log.Info("found in cache", "ref", c.(*unconfirmedBlocksCache).referenceCount)
		return nil
	}

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

	p.fullCache.Store(addr, &unconfirmedBlocksCache{
		blocks:         *list,
		currentEle:     list.Front(),
		referenceCount: 1,
	})

	return nil
}

func (p *UnconfirmedBlocksPool) ReleaseAccountInfoCache(addr types.Address) error {
	log := p.log.New("ReleaseAccountInfoCache", addr)
	v, ok := p.fullCache.Load(addr)
	if !ok {
		log.Info("no cache found")
		return nil
	}
	c := v.(*unconfirmedBlocksCache)
	if c.subReferenceCount() <= 0 {
		log.Info("cache found ref <= 0 delete cache")

		c.ResetCursor()
		p.fullCacheDeadTimer.Store(addr, time.AfterFunc(fullCacheExpireTime, func() {
			log.Info("cache delete")
			p.DeleteFullCache(addr)
		}))
		return nil
	}
	log.Info("after release", "ref", c.referenceCount)

	return nil
}

func (p *UnconfirmedBlocksPool) DeleteFullCache(address types.Address) {
	p.fullCache.Delete(address)
}

// todo support batch
func (p *UnconfirmedBlocksPool) WriteUnconfirmed(batch *leveldb.Batch, blockList []*vm_context.VmAccountBlock) error {
	p.log.Info("WriteUnconfirmed ")

	for _, v := range blockList {if v.AccountBlock.IsSendBlock() {
		if err := p.dbAccess.writeUnconfirmedMeta(batch, v.AccountBlock); err != nil {
			p.log.Error("writeUnconfirmedMeta", "error", err)
			return err
		}// add the gid-contractAddrList relationship
			var unsavedCache vmctxt_interface.UnsavedCache
			unsavedCache = v.VmContext.UnsavedCache()
			gidList := unsavedCache.ContractGidList()
			for _, v := range gidList {
		// todop.dbAccess.WriteContractAddrToGid(batch, *v.Gid(), *v.Addr())
			}
	} else {
		if err := p.dbAccess.deleteUnconfirmedMeta(batch, v.AccountBlock); err != nil {
			p.log.Error("deleteUnconfirmedMeta", "error", err)
			return err}
		}
	}
	// todo 确认写好之后 再更新
	p.updateCache(writeType, block)
	return nil
}

// DeleteUnRoad means to revert according to bifurcation
func (p *UnconfirmedBlocksPool) DeleteUnconfirmed(batch *leveldb.Batch, subLedger map[types.Address][]*ledger.AccountBlock) error {
	p.log.Info("DeleteUnconfirmed: revert")
	for _, blockList := range subLedger {
		for _, v := range blockList {
			if v.IsReceiveBlock() {
				sendBlock, err := p.dbAccess.Chain.GetAccountBlockByHash(&v.FromBlockHash)
				if err != nil {
					p.log.Error("GetAccountBlockByHash", "error", err)
					return err
				}
				if err := p.dbAccess.writeUnconfirmedMeta(batch, sendBlock); err != nil {
					p.log.Error("revert receiveBlock failed", "error", err)
					return err
				}
			} else {
				if err := p.dbAccess.deleteUnconfirmedMeta(batch, v); err != nil {
					p.log.Error("revert sendBlock failed", "error", err)
					return err
				}

				// delete the gid-contractAddrList relationship
				gidList := contracts.GetGidFromCreateContractData(v.Data)
				for _, v := range gidList {
					p.dbAccess.DeleteContractAddrFromGid(batch, *v.Gid(), *v.Addr())
				}
			}
		}
	}
	return nil
}

func (p *UnconfirmedBlocksPool) updateFullCache(writeType bool, block *ledger.AccountBlock) error {
	v, ok := p.fullCache.Load(block.ToAddress)
	fullCache := v.(*unconfirmedBlocksCache)
	// todo check == 0
	if !ok || fullCache.blocks.Len() == 0 {
		//p.log.Info("updateCache：no fullCache")
		return nil
	}

	if writeType {
		fullCache.addTx(block)
	} else {
		fullCache.rmTx(block)
	}

	return nil
}

// todo add mutex
func (p *UnconfirmedBlocksPool) updateSimpleCache(writeType bool, block *ledger.AccountBlock) error {

	value, ok := p.simpleCache.Load(block.ToAddress)
	if !ok {
		// p.log.Info("updateSimpleCache：no cache")
		return nil
	}
	simpleAccountInfo := value.(*CommonAccountInfo)

	tokenBalanceInfo, ok := simpleAccountInfo.TokenBalanceInfoMap[block.TokenId]
	if writeType {
		if ok {
			tokenBalanceInfo.TotalAmount.Add(&tokenBalanceInfo.TotalAmount, block.Amount)
			tokenBalanceInfo.Number += 1
		} else {
			// todo remove token info
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
	gid, err := p.dbAccess.Chain.GetContractGid(&block.AccountAddress)
	if err != nil {
		p.log.Error("NewSignalToWorker", "err", err)
		return
	}
	if gid != nil {
		p.contractListenerMutex.RLock()
		defer p.contractListenerMutex.RUnlock()
		if f, ok := p.newContractListener[*gid]; ok {
			f()
		}
	} else {
		p.commonTxListenerMutex.RLock()
		defer p.commonTxListenerMutex.RUnlock()
		if f, ok := p.newCommonTxListener[block.ToAddress]; ok {
			f()
		}
	}
}

func (p *UnconfirmedBlocksPool) GetUnconfirmedBlocks(index, num, count uint64, addr *types.Address) (blockList []*ledger.AccountBlock, err error) {
	return p.dbAccess.GetUnconfirmedBlocks(index, num, count, addr)
}

func (p *UnconfirmedBlocksPool) AddCommonTxLis(addr types.Address, f func()) {
	p.commonTxListenerMutex.Lock()
	defer p.commonTxListenerMutex.Unlock()
	p.newCommonTxListener[addr] = f
}

func (p *UnconfirmedBlocksPool) RemoveCommonTxLis(addr types.Address) {
	p.commonTxListenerMutex.Lock()
	defer p.commonTxListenerMutex.Unlock()
	delete(p.newCommonTxListener, addr)
}

func (p *UnconfirmedBlocksPool) AddContractLis(gid types.Gid, f func()) {
	p.contractListenerMutex.Lock()
	defer p.contractListenerMutex.Unlock()
	p.newContractListener[gid] = f
}

func (p *UnconfirmedBlocksPool) RemoveContractLis(gid types.Gid) {
	p.contractListenerMutex.Lock()
	defer p.contractListenerMutex.Unlock()
	delete(p.newContractListener, gid)
}
