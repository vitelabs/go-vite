package consensus

import (
	"testing"

	"sync"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
)

func Test(t *testing.T) {

	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	rw := chainRw{rw: c}

	wg := sync.WaitGroup{}

	closed := false

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !closed {
				addr, _ := types.HexToHash("fc08446111289c671fe1547f634afcde92fab289c11fe16380958305b2f379ad")
				list, _ := rw.rw.GetRegisterList(addr, types.SNAPSHOT_GID)
				r := make(map[types.Address]bool)
				for _, v := range list {
					if r[v.NodeAddr] || v.NodeAddr.String() == "vite_0000000000000000000000000000000000000000a4f3a0cb58" {
						t.Error(v.NodeAddr)
						closed = true
						return
					}
					r[v.NodeAddr] = true
				}
			}

		}()
	}

	wg.Wait()
}
