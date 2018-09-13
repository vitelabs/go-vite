package model

import (
	"container/list"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	oldledger "github.com/vitelabs/go-vite/ledger_old"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"sync"
	"time"
)

type UnconfirmedMeta struct {
	Gid     []byte
	Address types.Address
	Hash    types.Hash
}

type CommonAccountInfo struct {
	AccountAddress *types.Address
	TotalNumber    uint64
	TokenInfoMap   map[types.TokenTypeId]*TokenInfo
}

// pack the data for handler
type TokenInfo struct {
	Token       oldledger.Mintage
	TotalAmount big.Int
	Number      big.Int
}

//type Tx struct {
//	Hash   types.Hash
//	amount big.Int
//}

type unconfirmedBlocksCache struct {
	blocks         list.List
	lastUpdateTime time.Time
	currentPoint   *list.Element
	// lastReadTime   time.Time
}

func (c *unconfirmedBlocksCache) GetNextTx() *ledger.AccountBlock {
	if c.currentPoint == nil {
		return nil
	}

	block := c.currentPoint.Value.(*ledger.AccountBlock)
	c.currentPoint = c.currentPoint.Next()
	return block
}

func (c *unconfirmedBlocksCache) addTx(b *ledger.AccountBlock) {
	c.blocks.PushBack(b)
}

func (c *unconfirmedBlocksCache) rmTx(b *ledger.AccountBlock) {
	if b == nil {
		return
	}
	ele := c.blocks.Front()
	for ele != nil {
		next := ele.Next()
		if ele.Value.(*ledger.AccountBlock).Hash == b.Hash {
			c.blocks.Remove(ele)
		}
		ele = next
	}
}

// obtaining the account info from cache or db and manage the cache lifecycle
type UnconfirmedBlocksPool struct {
	caches map[types.Address]*unconfirmedBlocksCache

	cacheMutex sync.RWMutex
	dbAccess   *UAccess

	newCommonTxListener map[types.Address]func()
	newContractListener map[types.Gid]func()
	log                 log15.Logger
}

func NewAccInfoPool(dbAccess *UAccess) *UnconfirmedBlocksPool {
	return &UnconfirmedBlocksPool{
		caches:   make(map[types.Address]*unconfirmedBlocksCache),
		dbAccess: dbAccess,
		log:      log15.New("unconfirmed", "UnconfirmedBlocksPool"),
	}
}

func (p *UnconfirmedBlocksPool) Start() {

}

func (p *UnconfirmedBlocksPool) Stop() {

}

func (p *UnconfirmedBlocksPool) GetNextTx(address types.Address) *ledger.AccountBlock {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()
	c, ok := p.caches[address]
	if !ok {
		p.cacheMutex.RUnlock()
		return nil
	}
	return c.GetNextTx()

}

func (p *UnconfirmedBlocksPool) AcquireAccountInfoCache(address types.Address) error {
	p.cacheMutex.RLock()
	if _, ok := p.caches[address]; ok {
		p.cacheMutex.RUnlock()
		return nil
	}

	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	blocks, e := p.dbAccess.GetAllUnconfirmedBlocks(address)
	if e != nil {
		return e
	}
	list := list.New()
	for _, value := range blocks {
		list.PushBack(value)
	}
	p.caches[address] = &unconfirmedBlocksCache{
		blocks:         *list,
		currentPoint:   list.Front(),
		lastUpdateTime: time.Now(),
	}

	return nil
}

func (p *UnconfirmedBlocksPool) ReleaseAccountInfoCache(address types.Address) error {
	//  todo keep the cache in memory some time

	return nil
}

func (p *UnconfirmedBlocksPool) ClearAccountInfoCache(address types.Address) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	delete(p.caches, address)
}

func (p *UnconfirmedBlocksPool) WriteUnconfirmed(writeType bool, batch *leveldb.Batch, block *ledger.AccountBlock) error {
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
	if err := p.updateCache(writeType, block); err != nil {
		p.log.Error("updateCache", "error", err)
	}

	return nil
}

func (p *UnconfirmedBlocksPool) updateCache(writeType bool, block *ledger.AccountBlock) error {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()
	cache, ok := p.caches[block.ToAddress]
	if !ok || cache.blocks.Len() == 0 {
		p.log.Info("updateCache：no cache")
		return nil
	}

	if writeType {
		cache.addTx(block)
	} else {
		cache.rmTx(block)
	}

	return nil
}

func (p *UnconfirmedBlocksPool) NewSignalToWorker(block *ledger.AccountBlock) {
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

//func (access *UAccess) UpdateCommonAccInfo(writeType bool, block *ledger.AccountBlock) error {
//	tiMap, ok := access.commonAccountInfoMap[block.ToAddress]
//	if !ok {
//		access.log.Info("UpdateCommonAccInfo：no memory maintenance:",
//			"reason", "send-to address doesn't belong to current manager")
//		return nil
//	}
//	select {
//	case writeType == true:
//		ti, ok := tiMap.TokenInfoMap[block.TokenId]
//		if !ok {
//			token, err := access.Chain.GetTokenInfoById(&block.TokenId)
//			if err != nil {
//				return errors.New("func UpdateCommonAccInfo.GetByTokenId failed" + err.Error())
//			}
//			tiMap.TokenInfoMap[block.TokenId].Token = token.Mintage
//			tiMap.TokenInfoMap[block.TokenId].TotalAmount = *block.Amount
//		} else {
//			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
//		}
//
//	case writeType == false:
//		ti, ok := tiMap.TokenInfoMap[block.TokenId]
//		if !ok {
//			return errors.New("find no memory tokenInfo, so can't update when writeType is false")
//		} else {
//			if ti.TotalAmount.Cmp(block.Amount) == -1 {
//				return errors.New("conflict with the memory info, so can't update when writeType is false")
//			}
//			if ti.TotalAmount.Cmp(block.Amount) == 0 {
//				delete(tiMap.TokenInfoMap, block.TokenId)
//			} else {
//				ti.TotalAmount.Sub(&ti.TotalAmount, block.Amount)
//			}
//		}
//	}
//	return nil
//}
