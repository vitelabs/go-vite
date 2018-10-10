package model

import (
	"testing"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"time"
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
	pool := newOnroadBlocksPool()

	pool.AcquireFullOnroadBlocksCache()

}
