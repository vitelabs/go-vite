package net

import (
	net2 "net"
	"sync"

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

func (n *mockNet) Trace() {

}

func (n *mockNet) Stop() error {
	return nil
}

func (n *mockNet) ProtoData() []byte {
	return nil
}

func (n *mockNet) ReceiveHandshake(msg p2p.HandshakeMsg, protoData []byte, sender net2.Addr) (state interface{}, level p2p.Level, err error) {
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
	return NodeInfo{}
}

func mock(cfg Config) Net {
	peers := newPeerSet()

	feed := newBlockFeeder()

	receiver := &safeBlockNotifier{
		blockFeeder: feed,
		Verifier:    cfg.Verifier,
	}

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
		log:       netLog.New("module", "syncer"),
	}
	syncer.state = syncStateDone{syncer}

	return &mockNet{
		Config: &Config{
			Single: true,
		},
		chain:  cfg.Chain,
		syncer: syncer,
		fetcher: &fetcher{
			filter:   newFilter(),
			st:       0,
			receiver: receiver,
			policy: &fetchTarget{peers, func() bool {
				return true
			}},
			log:  netLog.New("module", "fetcher"),
			term: nil,
		},
		broadcaster: &broadcaster{
			peers:     peers,
			st:        SyncDone,
			verifier:  cfg.Verifier,
			feed:      feed,
			filter:    nil,
			store:     nil,
			mu:        sync.Mutex{},
			statistic: circle.NewList(records24h),
			log:       netLog.New("module", "broadcaster"),
		},
		BlockSubscriber: feed,
	}
}
