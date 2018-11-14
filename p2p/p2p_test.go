package p2p

import (
	"crypto/rand"
	"github.com/vitelabs/go-vite/p2p/block"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"testing"
	"time"
)

var blockUtil = block.New(blockPolicy)

func TestBlock(t *testing.T) {
	var id discovery.NodeID
	rand.Read(id[:])

	if blockUtil.Blocked(id[:]) {
		t.Fail()
	}

	blockUtil.Block(id[:])

	if !blockUtil.Blocked(id[:]) {
		t.Fail()
	}

	time.Sleep(blockMinExpired)
	if blockUtil.Blocked(id[:]) {
		t.Fail()
	}
}
