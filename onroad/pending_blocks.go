package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type callerPendingMap struct {
	taskAddr  *types.Address
	pmap      map[*types.Address][]*ledger.AccountBlock
	blackList map[*types.Address]bool
}

func newCallerPendingMap() *callerPendingMap {
	return &callerPendingMap{
		pmap:      make(map[*types.Address][]*ledger.AccountBlock, 0),
		blackList: make(map[*types.Address]bool, 0),
	}
}

func (p *callerPendingMap) isPendingMapNotSufficient() bool {
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
	for _, v := range p.pmap {
		if len(v) > 0 {
			return v[0]
		}
	}
	return nil
}

func (p *callerPendingMap) addPendingMap(sendBlock *ledger.AccountBlock) {
	if l, ok := p.pmap[&sendBlock.AccountAddress]; ok && l != nil {
		l = append(l, sendBlock)
	} else {
		new_l := make([]*ledger.AccountBlock, 0)
		new_l = append(new_l, sendBlock)
		p.pmap[&sendBlock.AccountAddress] = new_l
	}
}

func (p *callerPendingMap) deletePendingMap(caller *types.Address, sendHash *types.Hash) {
	if l, ok := p.pmap[caller]; ok && l != nil {
		for k, v := range l {
			if v.Hash == *sendHash {
				if k >= len(l)-1 {
					l = l[0:k]
				} else {
					l = append(l[0:k], l[k+1:]...)
				}
				break
			}
		}
	}
}

func (p *callerPendingMap) addCallerIntoBlackList(caller *types.Address) {
	p.blackList[caller] = true
}

func (p *callerPendingMap) existInBlackList(caller *types.Address) bool {
	if isExist, ok := p.blackList[caller]; ok {
		return isExist
	}
	return false
}
