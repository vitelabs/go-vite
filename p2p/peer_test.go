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

	p1 := NewPeer(vnode.ZERO, "hello", 100, "", 1, c1, 1, &mockProtocol{})

	var id vnode.NodeID
	id[0] = 1
	p2 := NewPeer(id, "world", 100, "", 1, c2, 1, &mockProtocol{})

	go func() {
		err := p1.WriteMsg(Msg{
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

	p1 := NewPeer(id1, "peer1", 100, "", 1, c1, 1, &mockProtocol{})
	p2 := NewPeer(id2, "peer2", 100, "", 1, c2, 1, &mockProtocol{})

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
			atomic.AddInt32(&w, -1)
		}()
	}

	wg.Wait()

	if w != 0 {
		b.Errorf("should not be %d", w)
	}
}

func BenchmarkWG(b *testing.B) {
	var wg sync.WaitGroup

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			wg.Add(1)
			wg.Done()
		}()
	}

	wg.Wait()
}
