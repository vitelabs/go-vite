package onroad

import (
	"container/list"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

const DefaultPullCount uint8 = 5

type inferiorState int

const (
	RETRY inferiorState = iota
	OUT
)

type callerPendingMap struct {
	pmap         map[types.Address]*list.List
	inferiorList map[types.Address]inferiorState

	addrMutex sync.RWMutex
}

func newCallerPendingMap() *callerPendingMap {
	return &callerPendingMap{
		pmap:         make(map[types.Address]*list.List),
		inferiorList: make(map[types.Address]inferiorState, 0),
	}
}

func (p *callerPendingMap) isPendingMapNotSufficient() bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if p.pmap == nil {
		return true
	}
	count := 0
	for _, list := range p.pmap {
		count += list.Len()
	}
	if count < int(DefaultPullCount) {
		return true
	}
	return false
}

func (p *callerPendingMap) Len() int {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	count := 0
	for _, list := range p.pmap {
		count += list.Len()
	}
	return count
}

func (p *callerPendingMap) getOnePending() *ledger.AccountBlock {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()

	for _, list := range p.pmap {
		if ele := list.Front(); ele != nil {
			front := ele.Value.(*ledger.AccountBlock)
			list.Remove(ele)
			return front
		}
	}
	return nil
}

func (p *callerPendingMap) addPendingMap(sendBlock *ledger.AccountBlock) (isExist bool) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()

	caller := sendBlock.AccountAddress

	value, ok := p.pmap[caller]
	if !ok || value == nil {
		l := list.New()
		l.PushBack(sendBlock)
		p.pmap[caller] = l
		return false
	}

	l := p.pmap[caller]
	var ele *list.Element
	if ele = l.Back(); ele != nil {
		pre := ele.Value.(*ledger.AccountBlock)
		if pre.Height == sendBlock.Height {
			return true
		}
		if pre.Height < sendBlock.Height {
			l.PushBack(sendBlock)
			return false
		}
	}
	l.Init()
	l.PushBack(sendBlock)
	return false
}

/*func (p *callerPendingMap) deletePendingMap(caller types.Address, sendHash *types.Hash) bool {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()

	list, ok := p.pmap[caller]
	if !ok || len(list) <= 0 {
		return false
	}
	for k, v := range p.pmap[caller] {
		if v.Hash != *sendHash {
			continue
		}
		if k >= len(p.pmap[caller])-1 {
			p.pmap[caller] = p.pmap[caller][0:k]
		} else {
			p.pmap[caller] = append(p.pmap[caller][0:k], p.pmap[caller][k+1:]...)
		}
		return true
	}
	return false
}*/

func (p *callerPendingMap) addCallerIntoInferiorList(caller types.Address, state inferiorState) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	p.inferiorList[caller] = state
	delete(p.pmap, caller)
}

func (p *callerPendingMap) existInInferiorList(caller types.Address) bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if _, ok := p.inferiorList[caller]; ok {
		return true
	}
	return false
}

func (p *callerPendingMap) removeFromInferiorList(caller types.Address) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	delete(p.inferiorList, caller)
}

func (p *callerPendingMap) isInferiorStateRetry(caller types.Address) bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if state, ok := p.inferiorList[caller]; ok && state == RETRY {
		return true
	}
	return false
}

func (p *callerPendingMap) isInferiorStateOut(caller types.Address) bool {
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	if state, ok := p.inferiorList[caller]; ok && state == OUT {
		return true
	}
	return false
}
