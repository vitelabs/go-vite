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

func TestBlock_F(t *testing.T) {
	var id discovery.NodeID
	rand.Read(id[:])

	for i := 0; i < blockCount-1; i++ {
		blockUtil.Block(id[:])
	}

	if !blockUtil.Blocked(id[:]) {
		t.Fail()
	}

	time.Sleep(blockMinExpired)
	if blockUtil.Blocked(id[:]) {
		t.Fail()
	}

	blockUtil.Block(id[:])

	time.Sleep(blockMinExpired)
	if !blockUtil.Blocked(id[:]) {
		t.Fail()
	}

	time.Sleep(blockMaxExpired)
	if blockUtil.Blocked(id[:]) {
		t.Fail()
	}
}
