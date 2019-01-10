package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/circle"
)

type mockNet struct {
	*Config
	*syncer
	*fetcher
	*broadcaster
	*receiver
}

func (n *mockNet) AddPlugin(plugin p2p.Plugin) {
}

func (n *mockNet) Info() *NodeInfo {
	return &NodeInfo{}
}

func (n *mockNet) Protocols() []*p2p.Protocol {
	return nil
}

func (n *mockNet) Start(svr p2p.Server) error {
	return nil
}

func (n *mockNet) Tasks() []*Task {
	return nil
}

func mock() Net {
	peers := newPeerSet()
	pool := &gid{}
	broadcaster := &broadcaster{
		peers:  peers,
		log:    log15.New("module", "mocknet/broadcaster"),
		statis: circle.NewList(records_24),
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
			pool:    &chunkPool{},
			running: 1,
		},
		fetcher: &fetcher{
			filter: filter,
			policy: &fetchPolicy{peers},
			pool:   pool,
			ready:  1,
		},
		broadcaster: broadcaster,
		receiver:    receiver,
	}
}
