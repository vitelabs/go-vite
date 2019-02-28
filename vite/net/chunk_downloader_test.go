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

func TestSplitChunk(t *testing.T) {
	from, to := mockFromTo()

	const batch = 300
	count := (to-from)/batch + 1

	fmt.Printf("from %d to %d, %d blocks, need %d chunks\n", from, to, to-from+1, count)

	cs := splitChunk(from, to, batch)

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

		if c[1] >= c[0]+batch {
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

	cs := splitChunk(from, to, 2)

	if uint64(len(cs)) != 1 {
		t.Fail()
	}

	if cs[0][0] != cs[0][1] || cs[0][0] != from {
		t.Fail()
	}
}

func TestSplitChunkCount(t *testing.T) {
	from := uint64(rand.Intn(1000))
	to := from + 1000

	cs := splitChunkCount(from, to, 50, 10)
	if len(cs) != 10 {
		t.Fatalf("to many chunks")
	}

	for _, c := range cs {
		t.Log(c[0], c[1])
	}
}

func TestSplitChunkMini(t *testing.T) {
	const batch = 333

	to := rand.Uint64()
	from := to - uint64(rand.Intn(batch-10))

	cs := splitChunk(from, to, batch)

	if uint64(len(cs)) != 1 {
		t.Fail()
	}

	if cs[0][0] != from || cs[0][1] != to {
		t.Fail()
	}
}

func TestChunkRequest_Done(t *testing.T) {
	var c chunkRequest
	c.from, c.to = mockFromTo()
	c.to = c.from + 100000

	const batch = 10
	cs := splitChunk(c.from, c.to, batch)

	var done bool
	var mid = len(cs) / 2
	for i := 0; i < mid; i++ {
		_, done = c.receive(ChunkResponse{
			From: cs[i][0],
			To:   cs[i][1],
		})
	}

	if done {
		t.Fail()
	}

	for i := mid; i < len(cs); i++ {
		_, done = c.receive(ChunkResponse{
			From: cs[i][0],
			To:   cs[i][1],
		})
	}

	if !done {
		t.Fail()
	}
}

func TestChunkRequest_Done2(t *testing.T) {
	var c chunkRequest
	c.from, c.to = mockFromTo()
	c.to = c.from + 100000

	const batch = 10
	cs := splitChunk(c.from, c.to, batch)

	var rest [][2]uint64
	var done bool
	var mid = len(cs) / 2
	for i := 0; i < mid; i++ {
		rest, done = c.receive(ChunkResponse{
			From: cs[i][0],
			To:   cs[i][1],
		})
	}

	if done {
		t.Fail()
	}

	for i := 0; i < len(rest); i++ {
		_, done = c.receive(ChunkResponse{
			From: rest[i][0],
			To:   rest[i][1],
		})
	}

	if !done {
		t.Fail()
	}
}

func BenchmarkChunkRequest_Receive(b *testing.B) {
	var c chunkRequest
	c.from, c.to = mockFromTo()
	c.to = c.from + 100000

	const batch = 10
	cs := splitChunk(c.from, c.to, batch)
	total := len(cs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := i % total
		c.receive(ChunkResponse{
			From: cs[j][0],
			To:   cs[j][1],
		})
	}
}
