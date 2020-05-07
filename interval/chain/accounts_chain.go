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

func (accts *accountsChain) one(addr string) *accountChain {
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

func (accts *accountsChain) oneFn(addr string, fn func(*accountChain)) {
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

func (accts *accountsChain) RollbackSnapshotBlocks(blocks []*common.SnapshotBlock) (map[string][]*common.AccountStateBlock, error) {
	accts.
		accts.rangeFn(func(acct *accountChain) bool {
		acct.RollbackSnapshotTo(block)
	})
}
