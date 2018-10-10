package model

import (
	"testing"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"time"
	"github.com/vitelabs/go-vite/common/types"
)

func newOnroadBlocksPool() *OnroadBlocksPool {
	chain := chain.NewChain(&config.Config{
		P2P:      nil,
		Miner:    nil,
		DataDir:  common.GoViteTestDataDir(),
		FilePort: 0,
		Topo:     nil,
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
