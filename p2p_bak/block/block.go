package block

import (
	"encoding/hex"
	"time"
)

type Record struct {
	T time.Time
	C int
}

type Policy func(t time.Time, count int) bool

type Block struct {
	records map[string]*Record
	policy  Policy
}

func New(policy Policy) *Block {
	return &Block{
		records: make(map[string]*Record),
		policy:  policy,
	}
}

func (b *Block) Block(buf []byte) {
	id := hex.EncodeToString(buf)
	if r, ok := b.records[id]; ok {
		r.T = time.Now()
		r.C++
	} else {
		b.records[id] = &Record{
			T: time.Now(),
			C: 1,
		}
	}
}

func (b *Block) UnBlock(buf []byte) {
	id := hex.EncodeToString(buf)
	delete(b.records, id)
}

func (b *Block) Blocked(buf []byte) bool {
	id := hex.EncodeToString(buf)

	if r, ok := b.records[id]; ok {
		return b.policy(r.T, r.C)
	}

	return false
}
