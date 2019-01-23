package net

import (
	"sync"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/circle"
)

type mockNet struct {
	*Config
	*syncer
	*fetcher
	*broadcaster
	*blockFeed
	chain Chain
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

func mock(cfg *Config) Net {
	peers := newPeerSet()
	pool := &gid{}

	feed := newBlockFeeder()

	return &mockNet{
		Config: &Config{
			Single: true,
		},
		chain: cfg.Chain,
		syncer: &syncer{
			state:   Syncdone,
			peers:   peers,
			pool:    &chunkPool{},
			running: 1,
		},
		fetcher: &fetcher{
			policy: &fp{peers},
			pool:   pool,
		},
		broadcaster: &broadcaster{
			peers:    peers,
			st:       Syncdone,
			verifier: cfg.Verifier,
			feed:     feed,
			filter:   nil,
			store:    nil,
			mu:       sync.Mutex{},
			statis:   circle.NewList(records_24),
			log:      log15.New("module", "mocknet/broadcaster"),
		},
	}
}
