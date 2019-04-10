package p2p

import (
	"net/http"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
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
	c1, c2 := MockPipe()
	id1, id2 := vnode.RandomNodeID(), vnode.RandomNodeID()

	var m1 = make(map[ProtocolID]peerProtocol)
	mp1 := &mockProtocol{}
	m1[mp1.ID()] = peerProtocol{
		Protocol: mp1,
	}

	var m2 = make(map[ProtocolID]peerProtocol)
	mp2 := &mockProtocol{}
	m2[mp2.ID()] = peerProtocol{
		Protocol: mp2,
	}

	p1 := NewPeer(id1, "peer1", 1, c1, 1, m1)
	p2 := NewPeer(id2, "peer2", 1, c2, 1, m2)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := p1.run(); err != nil {
			t.Error(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := p2.run(); err != nil {
			t.Error(err)
		}
	}()

	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()

	//time.Sleep(time.Second)
	//err := p1.Close(PeerQuitting)
	//if err != nil {
	//	t.Error(err)
	//}
}

func BenchmarkAtomic(b *testing.B) {
	var w int32

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			atomic.AddInt32(&w, 1)
			time.Sleep(time.Millisecond)
			atomic.AddInt32(&w, -1)
		}()
	}

	wg.Wait()

	if w != 0 {
		b.Errorf("should not be %d", w)
	}
}
