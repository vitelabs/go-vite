package net

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"testing"
)

var peerMap = newPeerSet()

func mockPeer() *peer {
	var id [32]byte

	crand.Read(id[:])

	return &peer{
		height: mrand.Uint64(),
		id:     hex.EncodeToString(id[:]),
	}
}

func TestPeerSet_SyncPeer(t *testing.T) {
	if peerMap.SyncPeer() != nil {
		t.Fail()
	}

	for i := 0; i < 10; i++ {
		p := mockPeer()
		peerMap.Add(p)
		fmt.Println(p.Height())
	}

	fmt.Println("mid", peerMap.SyncPeer().Height())
}

func TestPeerSet_SyncPeer2(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	if p = m.SyncPeer(); p != nil {
		t.Fail()
	}
}

func TestPeerSet_BestPeer(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	if p = m.BestPeer(); p != nil {
		t.Fail()
	}
}

func TestPeerSet_Get(t *testing.T) {
	var m = newPeerSet()
	var p Peer
	if p = m.Get("hello"); p != nil {
		t.Fail()
	}
}

func ExamplePeerSet_Get2() {
	var m = make(map[string]*peer)
	var p Peer = m["hello"]
	fmt.Println(p == nil)
	// Output: false
}

func ExamplePeerSet_Get3() {
	var m2 = make(map[string]Peer)
	var p Peer = m2["hello"]
	fmt.Println(p == nil)
	// Output: true
}
