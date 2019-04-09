package p2p

import (
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestP2P_run(t *testing.T) {
	c1, c2 := MockPipe()
	id1, id2 := vnode.RandomNodeID(), vnode.RandomNodeID()

	var m1 = make(map[ProtocolID]peerProtocol)
	mp1 := &mockProtocol{
		interval: 1 * time.Second,
	}
	m1[mp1.ID()] = peerProtocol{
		Protocol: mp1,
	}

	var m2 = make(map[ProtocolID]peerProtocol)
	mp2 := &mockProtocol{
		errFac: func() error {
			if rand.Intn(100000) < 10000 {
				return errors.New("mock handle error")
			}

			return nil
		},
		interval: 3 * time.Second,
	}
	m2[mp2.ID()] = peerProtocol{
		Protocol: mp2,
	}

	peer1 := NewPeer(id1, "peer1", 1, c1, 1, m1)
	peer2 := NewPeer(id2, "peer2", 1, c2, 1, m2)

	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			panic(err)
		}
	}()

	var p2p1 = &p2p{
		node: vnode.Node{
			ID: id1,
		},
		peers: newPeers(map[Level]int{
			Inbound:  10,
			Outbound: 10,
			Superior: 10,
		}),
		log: log15.New("module", "p2p_test"),
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		p2p1.register(peer2)
		_ = peer2.Close(PeerNetworkError)

		if p2p1.peers.has(peer2.ID()) {
			t.Errorf("%s should removed", peer2)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := peer1.run(); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}

//var blockUtil = block.New(blockPolicy)
//
//func TestBlock(t *testing.T) {
//	var id discovery.NodeID
//	rand.Read(id[:])
//
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	blockUtil.Block(id[:])
//
//	if !blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	time.Sleep(blockMinExpired)
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//}
//
//func TestBlock_F(t *testing.T) {
//	var id discovery.NodeID
//	rand.Read(id[:])
//
//	for i := 0; i < blockCount-1; i++ {
//		blockUtil.Block(id[:])
//	}
//
//	if !blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	time.Sleep(blockMinExpired)
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	blockUtil.Block(id[:])
//
//	time.Sleep(blockMinExpired)
//	if !blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//
//	time.Sleep(blockMaxExpired)
//	if blockUtil.Blocked(id[:]) {
//		t.Fail()
//	}
//}
