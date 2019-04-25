package onroad

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type OnRoadPool interface {
	WriteAccountBlock(block *ledger.AccountBlock) error
	DeleteAccountBlock(block *ledger.AccountBlock) error

	GetOnRoadFrontBlocks(addr types.Address) ([]*ledger.AccountBlock, error)
	GetOnRoadTotalNumByAddr(addr types.Address) (uint64, error)
}

type Chain interface {
	LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	IsContractAccount(address types.Address) (bool, error)
	IsGenesisAccountBlock(hash types.Hash) bool
}

type contractOnRoadPool struct {
	gid   types.Gid
	cache sync.Map //map[types.Address]*callerCache

	chain Chain
	log   log15.Logger
}

func NewContractOnRoadPool(gid types.Gid, chain Chain) OnRoadPool {
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
	p.log.Info("start loadOnRoad")
	contractMap, err := p.chain.LoadOnRoad(p.gid)
	if err != nil {
		return nil
	}
	// resort the map
	for contract, callerMap := range contractMap {
		cc := NewCallerCache()
		for caller, orList := range callerMap {
			if initErr := cc.initLoad(p.chain, caller, orList); initErr != nil {
				p.log.Error("loadOnRoad failed", "err", initErr, "caller", caller)
				return err
			}
		}
		p.cache.Store(contract, NewCallerCache())
	}
	p.log.Info("success loadOnRoad")
	return nil
}

func (p *contractOnRoadPool) GetOnRoadFrontBlocks(contract types.Address) ([]*ledger.AccountBlock, error) {
	cc, ok := p.cache.Load(contract)
	if !ok || cc == nil {
		return nil, nil
	}

	blockList := make([]*ledger.AccountBlock, 0)

	orList := cc.(*callerCache).getAllFrontTx()

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
	return uint64(cc.(*callerCache).Len()), nil
}

func (p *contractOnRoadPool) WriteAccountBlock(block *ledger.AccountBlock) error {
	mlog := p.log.New("method", "WriteAccountBlock")
	isWrite := true
	mlog.Info("WriteAccountBlock", "isSend", block.IsSendBlock())
	if block.IsSendBlock() {

		cc, exist := p.cache.Load(block.ToAddress)
		if !exist || cc == nil {
			cc, _ = p.cache.LoadOrStore(block.ToAddress, NewCallerCache())
		}

		caller := block.AccountAddress
		isCallerContract, err := p.chain.IsContractAccount(caller)
		if err != nil {
			return err
		}
		if isCallerContract {
			return ErrBlockTypeErr
		}

		or := &ledger.HashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		}
		if err := cc.(*callerCache).addTx(block.AccountAddress, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("write block-s: %v -> %v %v %v", block.AccountAddress, block.ToAddress, block.Height, block.Hash),
				"err", err)
			return ErrAddTxFailed
		}
	} else {
		// handle receive
		cc, exist := p.cache.Load(block.AccountAddress)
		if !exist {
			mlog.Error(fmt.Sprintf("write block-r: %v %v %v fromHash=%v", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash),
				"err", ErrLoadCallerCacheFailed)
			return ErrLoadCallerCacheFailed
		}
		fromBlock, err := p.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if fromBlock == nil {
			return errors.New("failed to find send")
		}
		caller := fromBlock.AccountAddress
		isCallerContract, err := p.chain.IsContractAccount(caller)
		if err != nil {
			return err
		}
		or := &ledger.HashHeight{
			Hash:   fromBlock.Hash,
			Height: fromBlock.Height,
		}
		if isCallerContract {
			completeBlock, err := p.chain.GetCompleteBlockByHash(fromBlock.Hash)
			if err != nil {
				return err
			}
			if completeBlock == nil {
				return errors.New("failed to find complete send's parent receive")
			}
			or.Height = completeBlock.Height // refer to its parent receive's height
		}
		if err := cc.(*callerCache).rmTx(caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("write block-r: %v %v %v fromHash=%v", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash),
				"err", err)
			return ErrRmTxFailed
		}

		// handle sendBlockList
		for _, subSend := range block.SendBlockList {
			isToAddrContract, err := p.chain.IsContractAccount(subSend.ToAddress)
			if err != nil {
				return err
			}
			if !isToAddrContract {
				continue
			}
			cc, exist := p.cache.Load(subSend.ToAddress)
			if !exist || cc == nil {
				cc, _ = p.cache.LoadOrStore(subSend.ToAddress, NewCallerCache())
			}
			or := &ledger.HashHeight{
				Hash:   subSend.Hash,
				Height: block.Height, // refer to its parent receive's height
			}
			if err := cc.(*callerCache).addTx(subSend.AccountAddress, isToAddrContract, or, isWrite); err != nil {
				mlog.Error(fmt.Sprintf("write block-s: %v -> %v %v %v", subSend.AccountAddress, subSend.ToAddress, block.Height, subSend.Hash),
					"err", err)
				return ErrAddTxFailed
			}
		}
	}
	return nil
}

func (p *contractOnRoadPool) DeleteAccountBlock(block *ledger.AccountBlock) error {
	mlog := p.log.New("method", "DeleteAccountBlock")
	isWrite := false
	mlog.Info("DeleteAccountBlock", "isSend", block.IsSendBlock())

	if block.IsSendBlock() {
		cc, exist := p.cache.Load(block.ToAddress)
		if !exist || cc == nil {
			mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", block.AccountAddress, block.ToAddress, block.Height, block.Hash),
				"err", ErrLoadCallerCacheFailed)
			return ErrLoadCallerCacheFailed
		}

		caller := block.AccountAddress
		isCallerContract, err := p.chain.IsContractAccount(caller)
		if err != nil {
			return err
		}
		if isCallerContract {
			return ErrBlockTypeErr
		}

		or := &ledger.HashHeight{
			Height: block.Height,
			Hash:   block.Hash,
		}
		if err := cc.(*callerCache).rmTx(caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", block.AccountAddress, block.ToAddress, block.Height, block.Hash),
				"err", err)
			return ErrRmTxFailed
		}
	} else {
		// revert receive
		cc, exist := p.cache.Load(block.AccountAddress)
		if !exist || cc == nil {
			cc, _ = p.cache.LoadOrStore(block.AccountAddress, NewCallerCache())
		}
		fromBlock, err := p.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if fromBlock == nil {
			return errors.New("failed to find send")
		}
		caller := fromBlock.AccountAddress
		isCallerContract, err := p.chain.IsContractAccount(caller)
		if err != nil {
			return err
		}
		or := &ledger.HashHeight{
			Hash:   fromBlock.Hash,
			Height: fromBlock.Height,
		}
		if isCallerContract {
			completeBlock, err := p.chain.GetCompleteBlockByHash(fromBlock.Hash)
			if err != nil {
				return err
			}
			if completeBlock == nil {
				return errors.New("failed to find complete send's parent receive")
			}
			or.Height = completeBlock.Height // refer to its parent receive's height
		}
		// revert how to ensure sequence contract's send ？？
		if err := cc.(*callerCache).addTx(caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("delete block-r: %v %v %v fromHash=%v", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash),
				"err", err)
			return ErrAddTxFailed
		}

		// revert sendBlockList
		for _, subSend := range block.SendBlockList {
			isToAddrContract, err := p.chain.IsContractAccount(subSend.ToAddress)
			if err != nil {
				return err
			}
			if !isToAddrContract {
				continue
			}
			cc, exist := p.cache.Load(subSend.ToAddress)
			if !exist || cc == nil {
				mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", subSend.AccountAddress, subSend.ToAddress, block.Height, subSend.Hash),
					"err", ErrLoadCallerCacheFailed)
				return ErrLoadCallerCacheFailed
			}
			or := &ledger.HashHeight{
				Hash:   subSend.Hash,
				Height: block.Height, // refer to its parent receive's height
			}
			if err := cc.(*callerCache).rmTx(subSend.ToAddress, isToAddrContract, or, isWrite); err != nil {
				mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", subSend.AccountAddress, subSend.ToAddress, block.Height, subSend.Hash),
					"err", err)
				return ErrRmTxFailed
			}
		}
	}
	return nil
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

func (cc *callerCache) initLoad(chain Chain, caller types.Address, orList []ledger.HashHeight) error {
	for k, _ := range orList {
		or := orList[k]
		// fmt.Printf("initLoad caller=%v height=%v, hash=%v\n", caller, or.Height, or.Hash)
		b, err := chain.GetAccountBlockByHash(or.Hash)
		if err != nil {
			return err
		}
		if b == nil {
			continue
		}
		if b.IsReceiveBlock() {
			return ErrBlockTypeErr
		}
		isContract, err := chain.IsContractAccount(b.AccountAddress)
		if err != nil {
			return err
		}
		if isContract {
			completeBlock, err := chain.GetCompleteBlockByHash(or.Hash)
			if err != nil {
				return err
			}
			if completeBlock == nil {
				return errors.New("failed to find complete send's parent receive")
			}
			or.Height = completeBlock.Height // refer to its parent receive's height
		}
		if err := cc.addTx(b.AccountAddress, isContract, &or, true); err != nil {
			return err
		}
	}
	return nil
}

func (cc *callerCache) getAllFrontTx() []*ledger.HashHeight {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	orList := make([]*ledger.HashHeight, 0)

	for _, l := range cc.cache {
		if ele := l.Front(); ele != nil {
			front := ele.Value.(*ledger.HashHeight)
			orList = append(orList, front)
		}
	}
	return orList
}

func (cc *callerCache) Len() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	count := 0
	for _, list := range cc.cache {
		count += list.Len()
	}
	return count
}

func (cc *callerCache) addTx(caller types.Address, isContract bool, or *ledger.HashHeight, isWrite bool) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	value, ok := cc.cache[caller]
	if !ok || value.Len() <= 0 {
		l := list.New()
		l.PushBack(or)
		cc.cache[caller] = l
		return nil
	}

	l := cc.cache[caller]
	for ele := l.Back(); ele != nil; ele = ele.Prev() {
		prev := ele.Value.(*ledger.HashHeight)
		if prev == nil {
			continue
		}
		if prev.Hash != or.Hash {
			if prev.Height > or.Height {
				continue
			}
			if prev.Height < or.Height {
				l.InsertAfter(or, ele)
				break
			}
			// prev.Height == or.Height
			if isContract {
				if !isWrite {
					continue
				}
				l.InsertAfter(or, ele)
				break
			}
			return errors.New("addTx fail, hash conflict at the same height")
		}
		return errors.New("addTx fail, duplicated")
	}
	return nil
}

func (cc *callerCache) rmTx(caller types.Address, isContract bool, or *ledger.HashHeight, isWrite bool) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	value, ok := cc.cache[caller]
	if !ok || value.Len() <= 0 {
		return errors.New("rmTx fail, callerList is nil")
	}

	l := cc.cache[caller]
	var ele *list.Element
	if isWrite {
		ele = l.Front()
		front := ele.Value.(*ledger.HashHeight)
		if front == nil || front.Hash != or.Hash {
			return errors.New("rmTx fail, write not at the most preferred")
		}
	} else {
		rmSuccess := false
		for ele = l.Back(); ele != nil; ele = ele.Prev() {
			prev := ele.Value.(*ledger.HashHeight)
			if prev == nil {
				continue
			}
			if prev.Hash == or.Hash {
				rmSuccess = true
				break
			}
		}
		if !rmSuccess {
			return errors.New("rmTx fail, can't find")
		}
	}
	l.Remove(ele)
	return nil
}
