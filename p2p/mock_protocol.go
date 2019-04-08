package p2p

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

type mockProtocol struct {
	mu    sync.Mutex
	peers map[vnode.NodeID]Peer
}

func (m *mockProtocol) Name() string {
	return "mock"
}

func (m *mockProtocol) ID() ProtocolID {
	return 255
}

func (m *mockProtocol) ProtoData() []byte {
	return nil
}

func (m *mockProtocol) ReceiveHandshake(msg HandshakeMsg, protoData []byte) (state interface{}, level Level, err error) {
	return nil, 1, nil
}

func (m *mockProtocol) Handle(msg Msg) error {
	fmt.Printf("code: %d, length: %d\n", msg.Code, len(msg.Payload))

	time.Sleep(10 * time.Millisecond)
	return msg.Sender.WriteMsg(msg)
}

func (m *mockProtocol) State() []byte {
	return nil
}

func (m *mockProtocol) SetState(state []byte, peer Peer) {
	return
}

func (m *mockProtocol) OnPeerAdded(peer Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.peers == nil {
		m.peers = make(map[vnode.NodeID]Peer)
	}

	if _, ok := m.peers[peer.ID()]; ok {
		return errors.New("peer existed")
	}

	m.peers[peer.ID()] = peer

	return nil
}

func (m *mockProtocol) OnPeerRemoved(peer Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.peers[peer.ID()]; ok {
		delete(m.peers, peer.ID())
		return nil
	}

	return errors.New("peer not existed")
}
