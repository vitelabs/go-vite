package net

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/net/vnode"
)

func TestPeerSet_Add(t *testing.T) {
	var m = newPeerSet()
	var p = &Peer{
		Id: vnode.RandomNodeID(),
	}

	// add first time
	if m.add(p) != nil {
		t.Fail()
	}

	// add repeatedly
	if m.add(p) == nil {
		t.Fail()
	}
}

func TestPeerSet_Del(t *testing.T) {
	var m = newPeerSet()
	var p = &Peer{
		Id: vnode.RandomNodeID(),
	}

	// should have no error
	if m.add(p) != nil {
		t.Fail()
	}

	p2, err := m.remove(p.Id)
	if err != nil {
		t.Fail()
	}

	if p != p2 {
		t.Fail()
	}

	if p2 = m.get(p.Id); p2 != nil {
		t.Fail()
	}
}

func TestPeerSet_SyncPeer(t *testing.T) {
	var m = newPeerSet()
	var p *Peer
	var err error

	if m.syncPeer() != nil {
		t.Fail()
	}

	if m.bestPeer() != nil {
		t.Fail()
	}

	for i := 0; i < 2; i++ {
		p = &Peer{
			Id:     vnode.RandomNodeID(),
			Height: uint64(i),
		}
		if err = m.add(p); err != nil {
			t.Errorf("failed to add peer: %v", err)
		}
	}

	// only two peers
	if p = m.syncPeer(); p.Height != 1 {
		t.Errorf("wrong sync peer height: %d", p.Height)
	}

	// reset
	m = newPeerSet()

	const total = 10
	for i := 0; i < total; i++ {
		p = &Peer{
			Id:     vnode.RandomNodeID(),
			Height: uint64(i),
		}
		if m.add(p) != nil {
			t.Fail()
		}
	}

	// the 1/3 peer
	if p = m.syncPeer(); p.Height != 6 {
		t.Errorf("wrong sync peer height: %d", p.Height)
	}
	if p = m.bestPeer(); p.Height != total-1 {
		t.Errorf("wrong best peer height: %d", p.Height)
	}
}

func TestPeerSet_Pick(t *testing.T) {
	var m = newPeerSet()
	var p *Peer

	const total = 10
	for i := 0; i < total; i++ {
		p = &Peer{
			Id:     vnode.RandomNodeID(),
			Height: uint64(i),
		}
		if m.add(p) != nil {
			t.Fail()
		}
	}

	const target = total / 2

	ps := m.pick(target)
	if len(ps) != total/2 {
		t.Errorf("wrong peer number: %d", len(ps))
	}
	for _, p = range ps {
		if p.Height < target {
			t.Errorf("wrong peer height: %d", p.Height)
		}
	}
}

func ExamplePeersSort() {
	var ps peers
	ps = append(ps, &Peer{
		Id:     vnode.RandomNodeID(),
		Height: 1,
	})
	ps = append(ps, &Peer{
		Id:     vnode.RandomNodeID(),
		Height: 2,
	})
	ps = append(ps, &Peer{
		Id:     vnode.RandomNodeID(),
		Height: 3,
	})

	sort.Sort(ps)
	fmt.Println(ps[0].Height)
	// Output:
	// 3
}

// test read write concurrently
func TestPeer_peers(t *testing.T) {
	var p = &Peer{
		m: make(map[peerId]struct{}),
	}

	go func() {
		patch := false
		const batch = 10
		for {
			ps := make([]peerConn, batch)
			for i := range ps {
				ps[i] = peerConn{
					id:  vnode.RandomNodeID().Bytes(),
					add: true,
				}
			}
			p.setPeers(ps, patch)
			patch = !patch
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		for {
			mp := p.peers()
			for k := range mp {
				delete(mp, k)
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()

	time.Sleep(5 * time.Second)
}

func TestPeer_peers2(t *testing.T) {
	const maxNeighbors = 200
	var total = rand.Intn(maxNeighbors)

	var pids = make([]vnode.NodeID, total)
	for i := range pids {
		pids[i] = vnode.RandomNodeID()
	}

	var m, m2 map[vnode.NodeID]struct{}

	m = make(map[vnode.NodeID]struct{})
	for i := 0; i < total/2; i++ {
		m[pids[i]] = struct{}{}
	}

	var p = &Peer{
		m: m,
	}

	p.setPeers(nil, true)
	m2 = p.peers()
	if len(m2) != len(p.m) {
		t.Errorf("wrong peers: %d", len(m2))
	}

	p.setPeers(nil, false)
	m2 = p.peers()
	if len(m2) != 0 {
		t.Errorf("wrong peers: %d", len(m2))
	}
	// reset
	p.m = m
	p.m2 = m

	idsPatch := make([]peerConn, total/4)
	var add = true
	for i := 0; i < len(idsPatch); i++ {
		idsPatch[i] = peerConn{
			id:  pids[i].Bytes(),
			add: add,
		}
		add = !add
	}

	old := len(m)
	p.setPeers(idsPatch, true)
	m2 = p.peers()
	should := old - len(idsPatch)/2

	if len(m2) != should {
		t.Errorf("should %d peers, but %d peers", should, len(m2))
	}
}
