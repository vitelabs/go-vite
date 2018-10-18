package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
)

type mockNet struct {
	*Config
	*syncer
	*fetcher
	*broadcaster
	*receiver
	*requestPool
}

func (n *mockNet) Info() *NodeInfo {
	return &NodeInfo{}
}

func (n *mockNet) Protocols() []*p2p.Protocol {
	return nil
}

func (n *mockNet) Start(svr *p2p.Server) error {
	return nil
}

func mock() Net {
	peers := newPeerSet()
	pool := newRequestPool(peers, nil)
	broadcaster := &broadcaster{
		peers: peers,
		log:   log15.New("module", "mocknet/broadcaster"),
	}
	filter := &filter{
		records: make(map[types.Hash]*record),
	}
	receiver := &receiver{
		ready:       0,
		sFeed:       newSnapshotBlockFeed(),
		aFeed:       newAccountBlockFeed(),
		broadcaster: broadcaster,
		filter:      filter,
	}

	return &mockNet{
		Config: &Config{
			Single: true,
		},
		syncer: &syncer{
			state:   Syncdone,
			feed:    newSyncStateFeed(),
			peers:   peers,
			pool:    pool,
			running: 1,
		},
		fetcher: &fetcher{
			filter: filter,
			peers:  peers,
			pool:   pool,
			ready:  1,
		},
		broadcaster: broadcaster,
		receiver:    receiver,
		requestPool: pool,
	}
}
