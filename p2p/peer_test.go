package p2p

import (
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestPeerMux(t *testing.T) {
	c1, c2 := MockPipe()

	p1 := NewPeer(vnode.ZERO, "hello", 100, types.Hash{1, 2, 3}, "", 1, c1, 1, &mockProtocol{})

	var id vnode.NodeID
	id[0] = 1
	p2 := NewPeer(id, "world", 100, types.Hash{1, 2, 3}, "", 1, c2, 1, &mockProtocol{})

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

	p1 := NewPeer(id1, "peer1", 100, types.Hash{1, 2, 3}, "", 1, c1, 1, &mockProtocol{})
	p2 := NewPeer(id2, "peer2", 100, types.Hash{1, 2, 3}, "", 1, c2, 1, &mockProtocol{})

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

func TestPeerMux_Disconnect(t *testing.T) {
	const addr = "localhost:9999"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	pub1, priv1, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	id1, _ := vnode.Bytes2NodeID(pub1)

	pub2, priv2, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	id2, _ := vnode.Bytes2NodeID(pub2)

	codecFactory := &transportFactory{
		minCompressLength: 100,
		readTimeout:       readMsgTimeout,
		writeTimeout:      writeMsgTimeout,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		hkr := &handshaker{
			version:     version,
			netId:       2,
			name:        "server",
			id:          id1,
			genesis:     types.Hash{4, 5, 6},
			fileAddress: nil,
			peerKey:     priv1,
			key:         nil,
			protocol:    &mockProtocol{}, // will be set when protocol registered
			log:         p2pLog.New("module", "handshaker"),
		}

		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		c := codecFactory.CreateCodec(conn)
		p, err := hkr.ReceiveHandshake(c)
		if err != nil {
			panic(err)
		}

		err = p.run()
		log.Println("client quit", err)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		hkr := &handshaker{
			version:     version,
			netId:       2,
			name:        "client",
			id:          id2,
			genesis:     types.Hash{4, 5, 6},
			fileAddress: nil,
			peerKey:     priv2,
			key:         nil,
			protocol:    &mockProtocol{}, // will be set when protocol registered
			log:         p2pLog.New("module", "handshaker"),
		}

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}

		c := codecFactory.CreateCodec(conn)

		p, err := hkr.InitiateHandshake(c, id1)
		if err != nil {
			panic(err)
		}

		p.Disconnect(PeerQuitting)
	}()

	wg.Wait()
}
