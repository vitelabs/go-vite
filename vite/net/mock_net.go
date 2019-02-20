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
			from:      0,
			to:        0,
			current:   0,
			aCount:    0,
			sCount:    0,
			state:     Syncdone,
			peers:     peers,
			pending:   0,
			responsed: 0,
			mu:        sync.Mutex{},
			fileMap:   make(map[filename]*fileRecord),
			chain:     cfg.Chain,
			eventChan: make(chan peerEvent),
			verifier:  cfg.Verifier,
			notifier:  newBlockFeeder(),
			fc:        nil,
			pool:      &chunkPool{},
			exec:      nil,
			curSubId:  0,
			subs:      make(map[int]SyncStateCallback),
			running:   1,
			term:      make(chan struct{}),
			log:       log15.New("module", "net/syncer"),
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
