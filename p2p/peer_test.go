package p2p

import (
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestPeerMux(t *testing.T) {
	c1, c2 := MockPipe()

	var m1 = make(map[ProtocolID]peerProtocol)
	mp1 := &mockProtocol{}
	m1[mp1.ID()] = peerProtocol{
		Protocol: mp1,
	}

	p1 := NewPeer(vnode.ZERO, "hello", 1, c1, 1, m1)

	var m2 = make(map[ProtocolID]peerProtocol)
	mp2 := &mockProtocol{}
	m2[mp2.ID()] = peerProtocol{
		Protocol: mp2,
	}

	var id vnode.NodeID
	id[0] = 1
	p2 := NewPeer(id, "world", 1, c2, 1, m2)

	go func() {
		err := p1.WriteMsg(Msg{
			pid:     mp1.ID(),
			Code:    0,
			Id:      0,
			Payload: []byte("hello"),
		})

		if err != nil {
			t.Error(err)
		}

		if err = p1.run(); err != nil {
			t.Error(err)
		}
	}()

	go func() {
		if err := p2.run(); err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(time.Second)
	err := p1.Close(PeerQuitting)
	if err != nil {
		t.Error(err)
	}
}

func TestPeerMux_Close(t *testing.T) {
	c1, _ := MockPipe()

	var m1 = make(map[ProtocolID]peerProtocol)
	mp1 := &mockProtocol{}
	m1[mp1.ID()] = peerProtocol{
		Protocol: mp1,
	}
	p1 := NewPeer(vnode.ZERO, "hello", 1, c1, 1, m1)

	go func() {
		if err := p1.run(); err != nil {
			t.Error(err)
		}
	}()

	time.Sleep(time.Second)
	err := p1.Close(PeerQuitting)
	if err != nil {
		t.Error(err)
	}
}
