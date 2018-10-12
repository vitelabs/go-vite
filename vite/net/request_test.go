package net

import (
	"fmt"
	"testing"
)

var peers = []*Peer{
	&Peer{height: 10000},
	&Peer{height: 7000},
	&Peer{height: 3000},
	&Peer{height: 8000},
	&Peer{height: 14000},
	&Peer{height: 18000},
	&Peer{height: 19000},
	&Peer{height: 20000},
}

func TestSplitSubLedger(t *testing.T) {
	cs := splitSubLedger(0, 20000, peers)
	for _, c := range cs {
		fmt.Println(c.from, c.to, c.peer.height)
	}
}

func TestSplitSubLedgerSmall(t *testing.T) {
	cs := splitSubLedger(0, 1000, peers)
	for _, c := range cs {
		fmt.Println(c.from, c.to, c.peer.height)
	}
}

func TestSplitChunk(t *testing.T) {
	cs := splitChunk(0, 30000)

	total := 30000/minBlocks + 1

	if uint64(len(cs)) != total {
		t.Fail()
	}

	for _, c := range cs {
		fmt.Println(c[0], c[1])
		if c[1] > c[0]+minBlocks {
			t.Fail()
		}
	}
}
