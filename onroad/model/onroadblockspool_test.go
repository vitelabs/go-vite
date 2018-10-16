package model

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/generator"
	"testing"
	"time"
)

func newOnroadBlocksPool() *OnroadBlocksPool {
	chain := chain.NewChain(&config.Config{
		P2P:     nil,
		DataDir: common.GoViteTestDataDir(),
	})

	uAccess := NewUAccess()
	uAccess.Init(chain)

	chain.Init()
	chain.Start()

	fullCacheExpireTime = 5 * time.Second
	simpleCacheExpireTime = 7 * time.Second

	return NewOnroadBlocksPool(uAccess)
}

func PrepareVite() chain.Chain {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	return c
}

var (
	p          = newOnroadBlocksPool()
	addrString = "vite_9832cec386721cbb458e23c01bd5cc3df9cca5cf2efade7ef2"
)

func TestOnroadBlocksPool_WriteOnroad(t *testing.T) {
	c := PrepareVite()

	addr, _ := types.HexToAddress(addrString)
	gen := generator.NewGenerator(c, nil, nil, addr)
	p.WriteOnroad()
}

func TestOnroadBlocksPool_RevertOnroad(t *testing.T) {

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
