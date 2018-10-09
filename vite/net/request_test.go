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
