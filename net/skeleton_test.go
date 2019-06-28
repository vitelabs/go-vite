package net

import (
	"fmt"
	"sync"
	"testing"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestSkeleton_Construct(t *testing.T) {
	peers := newPeerSet()
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})

	sk := newSkeleton(peers, new(gid))

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
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})
	peers.add(&mockPeer{
		id:     vnode.RandomNodeID(),
		height: 100,
	})

	sk := newSkeleton(peers, new(gid))

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
