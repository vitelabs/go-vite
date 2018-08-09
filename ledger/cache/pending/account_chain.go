package pending

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

var accountchainLog = log15.New("module", "ledger/cache/pending/account_chain")

type AccountchainPool struct {
	cache []*cacheItem
	lock  sync.Mutex
}

type cacheItem struct {
	block *ledger.AccountBlock
}

type processInterface interface {
	ProcessBlock(*ledger.AccountBlock)
}

func NewAccountchainPool(processor processInterface) *AccountchainPool {
	pool := AccountchainPool{}

	go func() {
		accountchainLog.Info("AccountchainPool: Start process block")
		turnInterval := time.Duration(2000)
		for {
			pool.lock.Lock()
			if len(pool.cache) <= 0 {
				pool.lock.Unlock()
				time.Sleep(turnInterval * time.Millisecond)
				continue
			}

			block := pool.cache[0].block
			pool.lock.Unlock()

			accountchainLog.Info("AccountchainPool: process 1 block")
			processor.ProcessBlock(block)
			accountchainLog.Info("AccountchainPool: process 1 block finish")

			pool.ClearHead()

		}
	}()

	return &pool
}

func (pool *AccountchainPool) ClearHead() {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if len(pool.cache) > 0 {
		pool.cache = pool.cache[1:]
	}
}

func (pool *AccountchainPool) Add(blocks []*ledger.AccountBlock) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	for _, block := range blocks {
		pool.cache = append(pool.cache, &cacheItem{
			block: block,
		})
	}

}
