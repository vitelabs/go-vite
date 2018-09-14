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

// obtaining the account info from cache or db and manage the cache lifecycle
type UnconfirmedBlocksCache struct {
	fullCache      map[types.Address]*unconfirmedBlocksCache
	fullCacheMutex sync.RWMutex

	simpleCache      map[types.Address]*CommonAccountInfo
	simpleCacheMutex sync.RWMutex

	dbAccess *UAccess

	newCommonTxListener map[types.Address]func()
	newContractListener map[types.Gid]func()

	log log15.Logger
}

func NewUnconfirmedBlocksPool(dbAccess *UAccess) *UnconfirmedBlocksCache {
	return &UnconfirmedBlocksCache{
		fullCache:   make(map[types.Address]*unconfirmedBlocksCache),
		simpleCache: make(map[types.Address]*CommonAccountInfo),
		dbAccess:    dbAccess,
		log:         log15.New("unconfirmed", "UnconfirmedBlocksCache"),
	}
}

func (p *UnconfirmedBlocksCache) Start() {

}

func (p *UnconfirmedBlocksCache) Stop() {

}

func (p *UnconfirmedBlocksCache) GetCommonAccountInfo(addr types.Address) (info *CommonAccountInfo, err error) {
	p.simpleCacheMutex.RLock()
	if c, ok := p.simpleCache[addr]; ok {
		p.simpleCacheMutex.RUnlock()
		return c, nil
	}
	p.simpleCacheMutex.RUnlock()

	p.fullCacheMutex.RLock()
	if f, ok := p.fullCache[addr]; ok {
		accountInfo := f.toCommonAccountInfo(p.dbAccess.chain.GetTokenInfoById)
		p.fullCacheMutex.RUnlock()
		return accountInfo, nil
	}
	p.fullCacheMutex.RUnlock()

	return p.dbAccess.GetCommonAccInfo(&addr)

}

func (p *UnconfirmedBlocksCache) GetNextTx(address types.Address) *ledger.AccountBlock {
	p.fullCacheMutex.RLock()
	defer p.fullCacheMutex.RUnlock()
	c, ok := p.fullCache[address]
	if !ok {
		p.fullCacheMutex.RUnlock()
		return nil
	}
	return c.GetNextTx()
}

func (p *UnconfirmedBlocksCache) AcquireAccountInfoCache(address types.Address) error {
	p.fullCacheMutex.RLock()
	if _, ok := p.fullCache[address]; ok {
		p.fullCacheMutex.RUnlock()
		return nil
	}

	p.fullCacheMutex.Lock()
	defer p.fullCacheMutex.Unlock()
	blocks, e := p.dbAccess.GetAllUnconfirmedBlocks(address)
	if e != nil {
		return e
	}

	list := list.New()
	for _, value := range blocks {
		list.PushBack(value)
	}

	p.fullCache[address] = &unconfirmedBlocksCache{
		blocks:         *list,
		currentEle:     list.Front(),
		lastUpdateTime: time.Now(),
	}

	return nil
}

func (p *UnconfirmedBlocksCache) ReleaseAccountInfoCache(address types.Address) error {
	//  todo keep the cache in memory some time

	return nil
}

func (p *UnconfirmedBlocksCache) ClearAccountInfoCache(address types.Address) {
	p.fullCacheMutex.Lock()
	defer p.fullCacheMutex.Unlock()
	delete(p.fullCache, address)
}

func (p *UnconfirmedBlocksCache) WriteUnconfirmed(writeType bool, batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if writeType { // add
		if err := p.dbAccess.writeUnconfirmedMeta(batch, block); err != nil {
			p.log.Error("writeUnconfirmedMeta", "error", err)
			return err
		}

		// fixme: @gx whether need to wait the block insert into chain and try the following
		p.NewSignalToWorker(block)
	} else { // delete
		if err := p.dbAccess.deleteUnconfirmedMeta(batch, block); err != nil {
			p.log.Error("deleteUnconfirmedMeta", "error", err)
			return err
		}
	}

	// fixme: @gx whether need to wait the block insert into chain and try the following
	p.updateCache(writeType, block)

	return nil
}

func (p *UnconfirmedBlocksCache) updateFullCache(writeType bool, block *ledger.AccountBlock) error {
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

func (p *UnconfirmedBlocksCache) updateSimpleCache(writeType bool, block *ledger.AccountBlock) error {
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
			token, err := p.dbAccess.chain.GetTokenInfoById(&block.TokenId)
			if err != nil {
				return errors.New("func UpdateCommonAccInfo.GetByTokenId failed" + err.Error())
			}
			simpleAccountInfo.TokenBalanceInfoMap[block.TokenId].Token = token.Mintage
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

func (p *UnconfirmedBlocksCache) updateCache(writeType bool, block *ledger.AccountBlock) {
	e := p.updateFullCache(writeType, block)
	if e != nil {
		p.log.Error("updateFullCache", "err", e)
	}

	e = p.updateSimpleCache(writeType, block)
	if e != nil {
		p.log.Error("updateSimpleCache", "err", e)
	}
}

func (p *UnconfirmedBlocksCache) NewSignalToWorker(block *ledger.AccountBlock) {
	if block.IsContractTx() {
		if f, ok := p.newContractListener[*block.Gid]; ok {
			f()
		}
	} else {
		if f, ok := p.newCommonTxListener[block.ToAddress]; ok {
			f()
		}
	}
}

func (p *UnconfirmedBlocksCache) AddCommonTxLis(addr types.Address, f func()) {
	p.newCommonTxListener[addr] = f
}

func (p *UnconfirmedBlocksCache) RemoveCommonTxLis(addr types.Address) {
	delete(p.newCommonTxListener, addr)
}

func (p *UnconfirmedBlocksCache) AddContractLis(gid types.Gid, f func()) {
	p.newContractListener[gid] = f
}

func (p *UnconfirmedBlocksCache) RemoveContractLis(gid types.Gid) {
	delete(p.newContractListener, gid)
}
