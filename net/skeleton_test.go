package net

import (
	"fmt"
	"sync"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/net/vnode"
)

func TestSkeleton_Construct(t *testing.T) {
	peers := newPeerSet()
	var err error
	for i := 0; i < 4; i++ {
		err = peers.add(&Peer{
			Id:     vnode.RandomNodeID(),
			Height: 100,
		})
		if err != nil {
			panic(err)
		}
	}

	sk := newSkeleton(peers, new(gid), make(map[types.Hash]struct{}))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			list := sk.construct([]*ledger.HashHeight{
				{
					Height: 99,
				},
			}, 100)

			fmt.Println(list)
		}()
	}

	wg.Wait()
}

func TestSkeleton_Reset(t *testing.T) {
	peers := newPeerSet()
	var err error
	for i := 0; i < 4; i++ {
		err = peers.add(&Peer{
			Id:     vnode.RandomNodeID(),
			Height: 100,
		})
		if err != nil {
			panic(err)
		}
	}

	sk := newSkeleton(peers, new(gid), make(map[types.Hash]struct{}))

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			if n%2 == 0 {
				list := sk.construct([]*ledger.HashHeight{
					{
						Height: 99,
					},
				}, 100)

				fmt.Println(list)
			} else {
				sk.reset()
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("done")
}
