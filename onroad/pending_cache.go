package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

type inferiorState int

const (
	RETRY inferiorState = iota
	OUT
)

type callerPendingMap struct {
	pmap         map[types.Address][]*ledger.AccountBlock
	InferiorList map[types.Address]inferiorState
	addrMutex    sync.RWMutex
}

func newCallerPendingMap() *callerPendingMap {
	return &callerPendingMap{
		pmap:         make(map[types.Address][]*ledger.AccountBlock, 0),
		InferiorList: make(map[types.Address]inferiorState, 0),
	}
}

func (p *callerPendingMap) isPendingMapNotSufficient() bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if p.pmap == nil {
		return true
	}
	var count int = 0
	for _, v := range p.pmap {
		count += len(v)
	}
	if count < int(DefaultPullCount)/2 {
		//fmt.Printf("isPendingMapNotSufficient count=%v\n", count)
		return true
	}
	return false
}

func (p *callerPendingMap) getPendingOnroad() *ledger.AccountBlock {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	for _, v := range p.pmap {
		if len(v) > 0 {
			return v[0]
		}
	}
	return nil
}

func (p *callerPendingMap) addPendingMap(sendBlock *ledger.AccountBlock) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	if list, ok := p.pmap[sendBlock.AccountAddress]; ok {
		for _, v := range list {
			if v.Hash == sendBlock.Hash {
				return
			}
		}
		p.pmap[sendBlock.AccountAddress] = append(p.pmap[sendBlock.AccountAddress], sendBlock)
	} else {
		new_l := make([]*ledger.AccountBlock, 0)
		new_l = append(new_l, sendBlock)
		p.pmap[sendBlock.AccountAddress] = new_l
	}
}

func (p *callerPendingMap) deletePendingMap(caller types.Address, sendHash *types.Hash) bool {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	if _, ok := p.pmap[caller]; ok {
		for k, v := range p.pmap[caller] {
			if v.Hash == *sendHash {
				if k >= len(p.pmap[caller])-1 {
					p.pmap[caller] = p.pmap[caller][0:k]
				} else {
					p.pmap[caller] = append(p.pmap[caller][0:k], p.pmap[caller][k+1:]...)
				}
				return true
			}
		}
	}
	return false
}

func (p *callerPendingMap) addCallerIntoInferiorList(caller types.Address, state inferiorState) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	p.InferiorList[caller] = state
	delete(p.pmap, caller)
}

func (p *callerPendingMap) existInInferiorList(caller types.Address) bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if _, ok := p.InferiorList[caller]; ok {
		return true
	}
	return false
}

func (p *callerPendingMap) removeFromInferiorList(caller types.Address) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	delete(p.InferiorList, caller)
}

func (p *callerPendingMap) isInferiorStateRetry(caller types.Address) bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if state, ok := p.InferiorList[caller]; ok && state == RETRY {
		return true
	}
	return false
}

func (p *callerPendingMap) isInferiorStateOut(caller types.Address) bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if state, ok := p.InferiorList[caller]; ok && state == OUT {
		return true
	}
	return false
}
