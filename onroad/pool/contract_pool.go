package onroad_pool

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sort"
	"sync"
)

type contractOnRoadPool struct {
	gid   types.Gid
	cache sync.Map //map[types.Address]*callerCache

	chain chainReader
	log   log15.Logger
}

func NewContractOnRoadPool(gid types.Gid, chain chainReader) OnRoadPool {
	or := &contractOnRoadPool{
		gid:   gid,
		chain: chain,
		log:   log15.New("contractOnRoadPool", gid),
	}
	if err := or.loadOnRoad(); err != nil {
		return nil
	}
	return or
}

func (p *contractOnRoadPool) loadOnRoad() error {
	p.log.Info("loadOnRoad from chain")
	contractMap, err := p.chain.LoadOnRoad(p.gid)
	if err != nil {
		return err
	}
	p.log.Info("start loadOnRoad into pool")
	// resort the map
	for contract, callerMap := range contractMap {
		cc, _ := p.cache.LoadOrStore(contract, NewCallerCache())
		for caller, orList := range callerMap {
			if initErr := cc.(*callerCache).initLoad(p.chain, caller, orList); initErr != nil {
				p.log.Error("loadOnRoad failed", "err", initErr, "caller", caller)
				return err
			}
		}
		if cc.(*callerCache).len() > 0 {
			p.log.Info(fmt.Sprintf("initLoad one caller, len=%v", cc.(*callerCache).len()), "contract", contract)
		}
	}
	p.log.Info("success loadOnRoad")
	return nil
}

func (p *contractOnRoadPool) IsFrontOnRoadOfCaller(orAddr, caller types.Address, hash types.Hash) (bool, error) {
	cc, ok := p.cache.Load(orAddr)
	if !ok || cc == nil {
		return false, ErrLoadCallerCacheFailed
	}
	or := cc.(*callerCache).getFrontTxByCaller(&caller)
	if or == nil || or.Hash != hash {
		return false, ErrCheckIsCallerFrontOnRoadFailed
	}
	return true, nil
}

func (p *contractOnRoadPool) GetFrontOnRoadBlocksByAddr(contract types.Address) ([]*ledger.AccountBlock, error) {
	cc, ok := p.cache.Load(contract)
	if !ok || cc == nil {
		return nil, nil
	}

	blockList := make([]*ledger.AccountBlock, 0)

	orList := cc.(*callerCache).getFrontTxOfAllCallers()

	for _, or := range orList {
		b, err := p.chain.GetAccountBlockByHash(or.Hash)
		if err != nil {
			return nil, err
		}
		if b == nil {
			continue
		}
		blockList = append(blockList, b)
	}
	return blockList, nil
}

func (p *contractOnRoadPool) GetOnRoadTotalNumByAddr(contract types.Address) (uint64, error) {
	cc, ok := p.cache.Load(contract)
	if !ok || cc == nil {
		return 0, nil
	}
	return uint64(cc.(*callerCache).len()), nil
}

func (p *contractOnRoadPool) InsertAccountBlocks(orAddr types.Address, blocks []*ledger.AccountBlock) error {
	mlog := p.log.New("method", "insertBlocks", "orAddr", orAddr, "len", len(blocks))
	isWrite := true
	onroadMap, err := p.ledgerBlockListToOnRoad(orAddr, blocks)
	if err != nil {
		return err
	}
	for _, pendingList := range onroadMap {
		sort.Sort(pendingList)
		for _, v := range pendingList {
			if v.block.IsSendBlock() {
				if err := p.insertOnRoad(v, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("write block-s: %v -> %v %v %v isWrite=%v", v.block.AccountAddress, v.block.ToAddress, v.block.Height, v.block.Hash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			} else {
				if err := p.deleteOnRoad(v, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("write block-r: %v %v %v fromHash=%v isWrite=%v", v.block.AccountAddress, v.block.Height, v.block.Hash, v.block.FromBlockHash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			}
		}
	}
	return nil
}

func (p *contractOnRoadPool) DeleteAccountBlocks(orAddr types.Address, blocks []*ledger.AccountBlock) error {
	mlog := p.log.New("method", "deleteBlocks", "orAddr", orAddr, "len", len(blocks))
	isWrite := false
	onroadMap, err := p.ledgerBlockListToOnRoad(orAddr, blocks)
	if err != nil {
		return err
	}
	for _, pendingList := range onroadMap {
		sort.Sort(pendingList)
		for i := pendingList.Len() - 1; i >= 0; i-- {
			v := pendingList[i]
			if v.block.IsSendBlock() {
				if err := p.deleteOnRoad(v, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v isWrite=%v", v.block.AccountAddress, v.block.ToAddress, v.block.Height, v.block.Hash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			} else {
				if err := p.insertOnRoad(v, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("delete block-r: %v %v %v fromHash=%v isWrite=%v", v.block.AccountAddress, v.block.Height, v.block.Hash, v.block.FromBlockHash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			}
		}
	}
	return nil
}

func (p *contractOnRoadPool) insertOnRoad(onroad *OnRoadBlock, isWrite bool) error {
	isCallerContract := types.IsContractAddr(onroad.caller)
	cc, exist := p.cache.Load(onroad.orAddr)
	if !exist || cc == nil {
		cc, _ = p.cache.LoadOrStore(onroad.orAddr, NewCallerCache())
	}
	if err := cc.(*callerCache).addTx(&onroad.caller, isCallerContract, &onroad.hashHeight, isWrite); err != nil {
		return err
	}
	return nil
}

func (p *contractOnRoadPool) deleteOnRoad(onroad *OnRoadBlock, isWrite bool) error {
	isCallerContract := types.IsContractAddr(onroad.caller)
	cc, exist := p.cache.Load(onroad.orAddr)
	if !exist || cc == nil {
		return ErrLoadCallerCacheFailed
	}
	if err := cc.(*callerCache).rmTx(&onroad.caller, isCallerContract, &onroad.hashHeight, isWrite); err != nil {
		return err
	}
	return nil
}

func (p *contractOnRoadPool) ledgerBlockListToOnRoad(orAddr types.Address, blocks []*ledger.AccountBlock) (map[types.Address]PendingOnRoadList, error) {
	onroadMap := make(map[types.Address]PendingOnRoadList)
	for _, b := range blocks {
		if b == nil {
			continue
		}
		onroad, err := LedgerBlockToOnRoad(p.chain, b)
		if err != nil {
			return nil, err
		}
		_, ok := onroadMap[onroad.caller]
		if !ok {
			onroadMap[onroad.caller] = make(PendingOnRoadList, 0)
		}
		onroadMap[onroad.caller] = append(onroadMap[onroad.caller], onroad)
	}
	return onroadMap, nil
}

type callerCache struct {
	cache map[types.Address]*list.List
	mu    sync.RWMutex
}

func NewCallerCache() *callerCache {
	return &callerCache{
		cache: make(map[types.Address]*list.List),
	}
}

func (cc *callerCache) initLoad(chain chainReader, caller types.Address, orList []ledger.HashHeight) error {
	orSortedList := make(onRoadList, 0)
	for k, _ := range orList {
		or := &orHashHeight{
			Height: orList[k].Height,
			Hash:   orList[k].Hash,
		}
		completeBlock, err := chain.GetCompleteBlockByHash(or.Hash)
		if err != nil {
			return err
		}
		if completeBlock == nil {
			return errors.New("failed to find complete block by hash")
		}
		if completeBlock.IsReceiveBlock() {
			or.Height = completeBlock.Height // refer to its parent receive's height
			for k, v := range completeBlock.SendBlockList {
				if v.Hash == or.Hash {
					idx := uint8(k)
					or.SubIndex = &idx
					break
				}
			}
		}
		orSortedList = append(orSortedList, or)
	}
	sort.Sort(orSortedList)
	for _, v := range orSortedList {
		isCallerContract := types.IsContractAddr(caller)
		if err := cc.addTx(&caller, isCallerContract, v, true); err != nil {
			return err
		}

	}
	return nil
}

func (cc *callerCache) getFrontTxOfAllCallers() []*orHashHeight {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	orList := make([]*orHashHeight, 0)

	for _, l := range cc.cache {
		if ele := l.Front(); ele != nil {
			front := ele.Value.(*orHashHeight)
			orList = append(orList, front)
		}
	}
	return orList
}

func (cc *callerCache) getFrontTxByCaller(caller *types.Address) *orHashHeight {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	value, exist := cc.cache[*caller]
	if !exist || value == nil {
		return nil
	}
	ele := cc.cache[*caller].Front()
	if ele == nil {
		return nil
	}
	return ele.Value.(*orHashHeight)
}

func (cc *callerCache) len() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	count := 0
	for _, l := range cc.cache {
		count += l.Len()
	}
	return count
}

func (cc *callerCache) addTx(caller *types.Address, isCallerContract bool, or *orHashHeight, isWrite bool) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	value, ok := cc.cache[*caller]
	if !ok || value.Len() <= 0 {
		l := list.New()
		l.PushFront(or)
		cc.cache[*caller] = l
		return nil
	}

	l := cc.cache[*caller]
	var ele *list.Element
	if isWrite {
		for ele = l.Back(); ele != nil; ele = ele.Prev() {
			prev := ele.Value.(*orHashHeight)
			if prev == nil {
				continue
			}
			if prev.Hash != or.Hash {
				if prev.Height > or.Height {
					continue
				}
				if prev.Height < or.Height {
					break
				}
				// prev.Height == or.Height
				if !isCallerContract || (prev.SubIndex == nil || or.SubIndex == nil) {
					return errors.New("addTx failed, hash conflict at the same height")
				}
				if *prev.SubIndex > *or.SubIndex {
					continue
				}
				if *prev.SubIndex < *or.SubIndex {
					break
				}
			}
			return errors.New("addTx failed, duplicated")
		}
		if ele == nil {
			l.PushFront(or)
		} else {
			l.InsertAfter(or, ele)
		}

	} else {
		for ele = l.Front(); ele != nil; ele = ele.Next() {
			next := ele.Value.(*orHashHeight)
			if next == nil {
				continue
			}
			if next.Hash != or.Hash {
				if next.Height < or.Height {
					continue
				}
				if next.Height > or.Height {
					break
				}
				// prev.Height == or.Height
				if !isCallerContract || (next.SubIndex == nil || or.SubIndex == nil) {
					return errors.New("addTx failed, hash conflict at the same height")
				}
				if *next.SubIndex < *or.SubIndex {
					continue
				}
				if *next.SubIndex > *or.SubIndex {
					break
				}
			}
			return errors.New("addTx failed, duplicated")
		}
		if ele == nil {
			l.PushBack(or)
		} else {
			l.InsertBefore(or, ele)
		}
	}

	return nil
}

func (cc *callerCache) rmTx(caller *types.Address, isCallerContract bool, or *orHashHeight, isWrite bool) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	value, ok := cc.cache[*caller]
	if !ok || value.Len() <= 0 {
		return errors.New("rmTx failed, callerList is nil")
	}

	l := cc.cache[*caller]
	var ele *list.Element
	if isWrite {
		ele = l.Front()
		front := ele.Value.(*orHashHeight)
		if front == nil || front.Hash != or.Hash {
			return errors.New("rmTx failed, write not at the most preferred")
		}
	} else {
		rmSuccess := false
		for ele = l.Back(); ele != nil; ele = ele.Prev() {
			prev := ele.Value.(*orHashHeight)
			if prev == nil {
				continue
			}
			if prev.Hash == or.Hash {
				rmSuccess = true
				break
			}
		}
		if !rmSuccess {
			return errors.New("rmTx failed, can't find the onroad")
		}
	}
	l.Remove(ele)
	return nil
}
