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
	chain Chain
	BlockSubscriber
}

func (n *mockNet) Stop() error {
	return nil
}

func (n *mockNet) ProtoData() []byte {
	return nil
}

func (n *mockNet) ReceiveHandshake(msg p2p.HandshakeMsg, protoData []byte) (state interface{}, level p2p.Level, err error) {
	return
}

func (n *mockNet) Start(svr p2p.P2P) error {
	return nil
}

func (n *mockNet) Name() string {
	return "mock_net"
}

func (n *mockNet) ID() p2p.ProtocolID {
	return ID
}

func (n *mockNet) Auth(input []byte) (output []byte) {
	return nil
}

func (n *mockNet) Handshake(their []byte) error {
	return nil
}

func (n *mockNet) Handle(msg p2p.Msg) error {
	return nil
}

func (n *mockNet) State() []byte {
	return nil
}

func (n *mockNet) SetState(state []byte, peer p2p.Peer) {
	return
}

func (n *mockNet) OnPeerAdded(peer p2p.Peer) error {
	return nil
}

func (n *mockNet) OnPeerRemoved(peer p2p.Peer) error {
	return nil
}

func (n *mockNet) Info() NodeInfo {
	return NodeInfo{
		PeerCount: 0,
		Latency:   nil,
	}
}

func mock(cfg Config) Net {
	peers := newPeerSet()

	feed := newBlockFeeder()

	syncer := &syncer{
		from:      0,
		to:        0,
		peers:     peers,
		mu:        sync.Mutex{},
		chain:     cfg.Chain,
		eventChan: make(chan peerEvent),
		curSubId:  0,
		subs:      make(map[int]SyncStateCallback),
		running:   1,
		term:      make(chan struct{}),
		downloader: newBatchDownloader(peers, &defaultSyncConnectionFactory{
			chain: cfg.Chain,
			peers: peers,
		}),
		log: log15.New("module", "net/syncer"),
	}
	syncer.state = syncStateDone{syncer}

	return &mockNet{
		Config: &Config{
			Single: true,
		},
		chain:  cfg.Chain,
		syncer: syncer,
		fetcher: &fetcher{
			policy: &fp{peers},
			idGen:  new(gid),
		},
		broadcaster: &broadcaster{
			peers:    peers,
			st:       SyncDone,
			verifier: cfg.Verifier,
			feed:     feed,
			filter:   nil,
			store:    nil,
			mu:       sync.Mutex{},
			statis:   circle.NewList(records24h),
			log:      log15.New("module", "mocknet/broadcaster"),
		},
		BlockSubscriber: feed,
	}
}
