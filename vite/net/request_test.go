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

// to is larger or equal than from
func mockFromTo() (from, to uint64) {
	from, to = mockU64(), mockU64()
	if from > to {
		from, to = to, from
	}

	return
}

func mockU64() uint64 {
	u := rand.Uint64()
	return u >> 40
}

func mockPeers(n int) (peers []*Peer) {
	var num int
	for {
		num = rand.Intn(n)
		if num > 0 {
			break
		}
	}
	fmt.Printf("mock %d peers\n", num)

	for i := 0; i < num; i++ {
		peers = append(peers, &Peer{ID: RandStringRunes(8), height: mockU64()})
	}

	return peers
}

func TestSplitSubLedger(t *testing.T) {
	peers := mockPeers(200)

	var to uint64
	for _, peer := range peers {
		if peer.height > to {
			to = peer.height
		}
	}

	var from uint64 = 0

	cs := splitSubLedger(from, to, peers)
	fmt.Printf("from %d to %d split %d ledger pieces\n", from, to, len(cs))

	low := from
	for _, c := range cs {
		if low != c.from {
			t.Fatalf("ledger piece is not coherent: %d, %d - %d @%d", low, c.from, c.to, c.peer.height)
		}

		if c.to < c.from {
			t.Fatalf("ledger piece from is larger than to: %d - %d @%d", c.from, c.to, c.peer.height)
		}

		if c.to > c.peer.height {
			t.Fatalf("ledger piece from out of peer: %d - %d @%d", c.from, c.to, c.peer.height)
		}

		low = c.to + 1
	}

	if low != to+1 {
		t.Fatalf("ledger pieces is not compelete, %d %d", low, to)
	}
}

func TestSplitChunk(t *testing.T) {
	from, to := mockFromTo()

	count := (to-from)/minBlocks + 1

	fmt.Printf("from %d to %d, %d blocks, need %d chunks\n", from, to, to-from+1, count)

	cs := splitChunk(from, to)

	if uint64(len(cs)) != count {
		t.Fail()
	}

	low := from
	for _, c := range cs {
		if c[0] != low {
			t.Fatalf("chunk is not coherent: %d, %d - %d", low, c[0], c[1])
		}

		if c[1] < c[0] {
			t.Fatalf("chunk from is larger than to: %d - %d", c[0], c[1])
		}

		if c[1] >= c[0]+minBlocks {
			t.Fatalf("chunk is too large: %d - %d", c[0], c[1])
		}

		low = c[1] + 1
	}

	if low != to+1 {
		t.Fatalf("chunks is not compelete, %d %d", low, to)
	}
}

func TestSplitChunkOne(t *testing.T) {
	to := rand.Uint64()
	from := to

	cs := splitChunk(from, to)

	if uint64(len(cs)) != 1 {
		t.Fail()
	}

	if cs[0][0] != cs[0][1] || cs[0][0] != from {
		t.Fail()
	}
}

//func TestU64ToDuration(t *testing.T) {
//	u := rand.Uint64()
//	u64ToDuration(u)
//}
