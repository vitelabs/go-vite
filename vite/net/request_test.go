package net

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func mockPeers() (peers []*Peer) {
	var num int
	for {
		num = rand.Intn(65535)
		if num > 0 {
			break
		}
	}
	fmt.Printf("mock %d peers\n", num)

	for i := 0; i < num; i++ {
		peers = append(peers, &Peer{ID: RandStringRunes(8), height: rand.Uint64()})
	}

	return peers
}

// to is larger or equal than from
func mockFromTo() (from, to uint64) {
	from, to = rand.Uint64(), rand.Uint64()
	if from > to {
		from, to = to, from
	}

	return
}

func TestSplitSubLedger(t *testing.T) {
	peers := mockPeers()

	var to uint64
	for _, peer := range peers {
		if peer.height > to {
			to = peer.height
		}
	}
	fmt.Println("to", to)

	var from uint64
	for {
		from = rand.Uint64()
		if from < to {
			break
		}
	}
	fmt.Println("from", from)

	cs := splitSubLedger(from, to, peers)
	fmt.Printf("split %d ledger pieces\n", len(cs))

	low := from
	for i, c := range cs {
		fmt.Printf("ledger piece %d: %d - %d %d blocks @%d\n", i, c.from, c.to, c.to-c.from+1, c.peer.height)

		if low != c.from {
			t.Fail()
		}

		if c.to < c.from {
			t.Fail()
		}

		if c.to > c.peer.height {
			t.Fail()
		}

		low = c.to + 1
	}

	if low != to+1 {
		t.Fail()
	}
}

func TestSplitChunk(t *testing.T) {
	to := rand.Uint64()
	from := to - uint64(rand.Intn(1000000000))

	count := (to-from)/minBlocks + 1

	fmt.Printf("from %d to %d, %d blocks, need %d chunks\n", from, to, from-to+1, count)

	cs := splitChunk(from, to)
	fmt.Printf("split %d chunks\n", len(cs))

	if uint64(len(cs)) != count {
		t.Fail()
	}

	low := from
	for i, c := range cs {
		fmt.Printf("chunk %d: %d - %d %d blocks\n", i, c[0], c[1], c[1]-c[0]+1)

		if c[0] != low {
			t.Fail()
		}

		if c[1] < c[0] {
			t.Fail()
		}

		if c[1] >= c[0]+minBlocks {
			t.Fail()
		}

		low = c[1] + 1
	}

	if low != to+1 {
		t.Fail()
	}
}

func TestSplitChunkOne(t *testing.T) {
	to := rand.Uint64()
	from := to
	fmt.Printf("from %d to %d, %d blocks, need %d chunks\n", from, to, from-to+1, 1)

	cs := splitChunk(from, to)

	if uint64(len(cs)) != 1 {
		t.Fail()
	}

	if cs[0][0] != cs[0][1] || cs[0][0] != from {
		t.Fail()
	}
}
