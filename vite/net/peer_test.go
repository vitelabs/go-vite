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
	if m.SyncPeer() != nil {
		t.Fail()
	}
}

func TestPeerSet_BestPeer(t *testing.T) {
	var m = newPeerSet()
	if m.BestPeer() != nil {
		t.Fail()
	}
}
