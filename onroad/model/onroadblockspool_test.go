package model

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"testing"
	"time"
)

func newOnroadBlocksPool() *OnroadBlocksPool {
	chain := chain.NewChain(&config.Config{
		P2P:      nil,
		DataDir:  common.GoViteTestDataDir(),
	})
	chain.Init()
	chain.Start()
	fullCacheExpireTime = 5 * time.Second
	simpleCacheExpireTime = 7 * time.Second

	return NewOnroadBlocksPool(NewUAccess(chain))
}

func TestOnroadBlocksPool_AcquireFullOnroadBlocksCache(t *testing.T) {
	addr, _, _ := types.CreateAddress()
	pool := newOnroadBlocksPool()
	for i := 0; i < 10; i++ {
		go func() {
			pool.AcquireFullOnroadBlocksCache(addr)
			pool.AcquireFullOnroadBlocksCache(addr)
			pool.ReleaseFullOnroadBlocksCache(addr)
		}()
		go func() {
			pool.AcquireFullOnroadBlocksCache(addr)
			pool.ReleaseFullOnroadBlocksCache(addr)
			pool.ReleaseFullOnroadBlocksCache(addr)
		}()
	}

	time.Sleep(20 * time.Second)

}
