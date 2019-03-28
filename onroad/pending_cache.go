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
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	var count int = 0
	for _, v := range p.pmap {
		count += len(v)
	}
	if count < int(DefaultPullCount)/2 {
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
	if l, ok := p.pmap[sendBlock.AccountAddress]; ok && l != nil {
		l = append(l, sendBlock)
	} else {
		new_l := make([]*ledger.AccountBlock, 0)
		new_l = append(new_l, sendBlock)
		p.pmap[sendBlock.AccountAddress] = new_l
	}
}

func (p *callerPendingMap) deletePendingMap(caller types.Address, sendHash *types.Hash) bool {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	if l, ok := p.pmap[caller]; ok && l != nil {
		for k, v := range l {
			if v.Hash == *sendHash {
				if k >= len(l)-1 {
					l = l[0:k]
				} else {
					l = append(l[0:k], l[k+1:]...)
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
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
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
