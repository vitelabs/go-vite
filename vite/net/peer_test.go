package net

import (
	"testing"
)

var peerMap = newPeerSet()

//func mockPeer() *peer {
//	var id [32]byte
//
//	crand.Read(id[:])
//
//	return &peer{
//		Peer:        nil,
//		peerMap:     sync.Map{},
//		knownBlocks: nil,
//		errChan:     nil,
//		once:        sync.Once{},
//		log:         nil,
//	}
//}

func TestPeerSet_SyncPeer(t *testing.T) {
	if peerMap.syncPeer() != nil {
		t.Fail()
	}

	//for i := 0; i < 10; i++ {
	//	p := mockPeer()
	//	peerMap.Add(p)
	//	fmt.Println(p.Height())
	//}
	//
	//fmt.Println("mid", peerMap.syncPeer().Height())
}

func TestPeerSet_SyncPeer2(t *testing.T) {
	var m = newPeerSet()
	if m.syncPeer() != nil {
		t.Fail()
	}
}

func TestPeerSet_BestPeer(t *testing.T) {
	var m = newPeerSet()
	if m.bestPeer() != nil {
		t.Fail()
	}
}
