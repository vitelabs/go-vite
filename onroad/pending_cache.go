package onroad

import (
	"container/list"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

const defaultPullCount uint8 = 5

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
	if count < int(defaultPullCount) {
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
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()

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

func (p *callerPendingMap) clearPendingMap() {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	p.pmap = make(map[types.Address]*list.List)
}

func (p *callerPendingMap) addIntoInferiorList(caller types.Address, state inferiorState) {
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	p.inferiorList[caller] = state
	delete(p.pmap, caller)
}

func (p *callerPendingMap) releaseCallerByState(state inferiorState) int {
	var count int
	p.addrMutex.Lock()
	defer p.addrMutex.Unlock()
	for caller, s := range p.inferiorList {
		if s == state {
			delete(p.inferiorList, caller)
			count++
		}
	}
	return count
}

func (p *callerPendingMap) lenOfCallersByState(state inferiorState) int {
	var count int
	p.addrMutex.RLock()
	defer p.addrMutex.RUnlock()
	for _, s := range p.inferiorList {
		if s == state {
			count++
		}
	}
	return count

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
