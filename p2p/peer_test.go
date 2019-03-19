package p2p

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/discovery"
)

// CheckWait will be blocked if peerNum is larger than minPeers
func TestPeerSet_CheckWait(t *testing.T) {
	const minPeers = 4
	var ps = NewPeerSet(minPeers)

	var i int
	go func() {
		for {
			ps.CheckWait()
			time.Sleep(time.Second)
			i++
		}
	}()

	time.Sleep(time.Second)
	var p *Peer
	for j := 0; j < minPeers; j++ {
		var id discovery.NodeID
		rand.Read(id[:])
		p = &Peer{
			ts: &transport{
				remoteID: id,
			},
		}

		ps.Add(p)

		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)
	if i > minPeers+1 {
		fmt.Println(i)
		t.Fail()
	}

	ps.Del(p)

	time.Sleep(2 * time.Second)
	if i != minPeers+3 {
		fmt.Println(i)
		t.Fail()
	}
}

// DisconnectAll will release CheckWait
func TestPeerSet_CheckWait2(t *testing.T) {
	const minPeers = 4
	var ps = NewPeerSet(minPeers)

	for j := 0; j < minPeers; j++ {
		var id discovery.NodeID
		rand.Read(id[:])
		ps.Add(&Peer{
			ts: &transport{
				remoteID: id,
			},
			disc: make(chan DiscReason, 1),
		})
	}

	ch := make(chan int)

	go func() {
		ps.CheckWait()
		ch <- 5
	}()

	ps.DisconnectAll()

	if i := <-ch; i != 5 {
		t.Fail()
	}
}
