package chain

import (
	"sync"

	"github.com/vitelabs/go-vite/interval/common"

	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/store"
)

type accountsChain struct {
	accounts sync.Map

	store    store.BlockStore
	listener face.ChainListener

	mu sync.Mutex // account chain init

	unconfirmed []*common.AccountHashH
}

func newAccountsChain(store store.BlockStore, listener face.ChainListener) *accountsChain {
	return &accountsChain{
		store:    store,
		listener: listener,
	}
}

func (accts *accountsChain) one(addr common.Address) *accountChain {
	chain, ok := accts.accounts.Load(addr)
	if !ok {
		c := newAccountChain(addr, accts.listener, accts.store)
		accts.mu.Lock()
		defer accts.mu.Unlock()
		accts.accounts.Store(addr, c)
		chain, _ = accts.accounts.Load(addr)
	}
	return chain.(*accountChain)
}

func (accts *accountsChain) oneFn(addr common.Address, fn func(*accountChain)) {
	chain, ok := accts.accounts.Load(addr)
	if !ok {
		c := newAccountChain(addr, accts.listener, accts.store)
		accts.mu.Lock()
		defer accts.mu.Unlock()
		accts.accounts.Store(addr, c)
		chain, _ = accts.accounts.Load(addr)
	}
	fn(chain.(*accountChain))
}

func (accts *accountsChain) rangeFn(fn func(acct *accountChain) bool) {
	accts.accounts.Range(func(key, chain interface{}) bool {
		return fn(chain.(*accountChain))
	})
}

func (accts *accountsChain) RollbackSnapshotBlocks(blocks []*common.SnapshotBlock) (map[common.Address][]*common.AccountStateBlock, error) {
	points := make(map[common.Address]*common.SnapshotPoint)
	for _, b := range blocks {
		for _, a := range b.Accounts {
			ac, ok := points[a.Addr]
			if !ok || ac.AccountHeight < a.Height {
				ac.AccountHeight = a.Height
				ac.AccountHash = a.Hash
				ac.SnapshotHeight = b.Height()
				ac.SnapshotHash = b.Hash()
				continue
			}
		}
	}
	var err error
	result := make(map[common.Address][]*common.AccountStateBlock)
	accts.rangeFn(func(acct *accountChain) bool {
		unconfirmed, e1 := acct.RollbackUnconfirmed()
		if e1 != nil {
			err = e1
			return false
		}
		result[acct.address] = append(result[acct.address], unconfirmed...)
		point, ok := points[acct.address]
		if !ok {
			return true
		}
		acctBlocks, e := acct.RollbackSnapshotPointTo(point)
		if e != nil {
			err = e
			return false
		}
		result[acct.address] = append(result[acct.address], acctBlocks...)
		return true
	})
	return result, err
}
