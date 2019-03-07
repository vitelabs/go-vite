package net

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"sort"
	"testing"
)

func mockPeer() *peer {
	var id [32]byte

	crand.Read(id[:])

	return &peer{
		height: mrand.Uint64(),
		id:     hex.EncodeToString(id[:]),
	}
}

func TestPeerSet_Add(t *testing.T) {
	var m = newPeerSet()
	var p *peer
	// should have error
	if m.Add(p) == nil {
		t.Fail()
	}
	if m.Count() != 0 {
		t.Fail()
	}

	p = mockPeer()
	if m.Add(p) != nil {
		t.Fail()
	}
	if m.Count() != 1 {
		t.Fail()
	}
}

func TestPeerSet_Del(t *testing.T) {
	var m = newPeerSet()
	var p = mockPeer()

	// should have no error
	if m.Add(p) != nil {
		t.Fail()
	}

	m.Del(p)

	var p2 Peer
	if p2 = m.Get(p.id); p2 != nil {
		t.Fail()
	}
	if m.Count() != 0 {
		t.Fail()
	}
}

type heights []uint64

func (h heights) Len() int {
	return len(h)
}

func (h heights) Less(i, j int) bool {
	return h[i] < h[j]
}

func (h heights) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func TestPeerSet_SyncPeer(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	if p = m.SyncPeer(); p != nil {
		t.Fail()
	}

	var hs heights

	for i := 0; i < 10; i++ {
		p2 := mockPeer()
		m.Add(p2)
		hs = append(hs, p2.height)
	}

	sort.Sort(hs)
	height := hs[len(hs)/2]

	if m.SyncPeer().Height() != height {
		t.Fail()
	}
}

func TestPeerSet_BestPeer(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	if p = m.BestPeer(); p != nil {
		t.Fail()
	}

	var hs heights

	for i := 0; i < 10; i++ {
		p2 := mockPeer()
		m.Add(p2)
		hs = append(hs, p2.height)
	}

	sort.Sort(hs)

	height := hs[len(hs)-1]
	if m.BestPeer().Height() != height {
		t.Fail()
	}
}

func TestPeerSet_Pick(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	if p = m.BestPeer(); p != nil {
		t.Fail()
	}

	var hs heights

	for i := 0; i < 10; i++ {
		p2 := mockPeer()
		m.Add(p2)
		hs = append(hs, p2.height)
	}

	sort.Sort(hs)

	height := hs[len(hs)-1]
	if ps := m.Pick(height); len(ps) != 1 {
		t.Fail()
	}

	mid := len(hs) / 2
	height = hs[mid]
	if ps := m.Pick(height); len(ps) != len(hs)-mid {
		t.Fail()
	}
}

func ExamplePeerSet_Get() {
	var m1 = newPeerSet()
	var p1 = m1.Get("hello")

	var m2 = make(map[string]Peer)
	var p2 = m2["hello"]

	var m3 = make(map[string]*peer)
	var p3 Peer = m3["hello"]

	fmt.Println(p1 == nil)
	fmt.Println(p2 == nil)
	fmt.Println(p3 == nil)
	// Output:
	// true
	// true
	// false
}
