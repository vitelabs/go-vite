package sync_cache

import (
	"crypto/rand"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func TestCacheItem_Serialize(t *testing.T) {
	var c = cacheItem{
		Segment: interfaces.Segment{
			From:     101,
			To:       1000,
			Hash:     types.Hash{},
			PrevHash: types.Hash{},
			Points:   nil,
		},
		done:     true,
		verified: true,
		filename: "chunk101_1000",
		size:     1837,
	}

	rand.Read(c.Hash[:])
	rand.Read(c.PrevHash[:])

	data, err := c.Serialize()
	if err != nil {
		panic(err)
	}

	var c2 = &cacheItem{}
	err = c2.DeSerialize(data)
	if err != nil {
		panic(err)
	}

	if false == c.Segment.Equal(c2.Segment) {
		t.Errorf("different segment")
	}
	for i, p := range c.Points {
		if c2.Points[i].Height != p.Height || c2.Points[i].Hash != p.Hash {
			t.Errorf("different point")
		}
	}
	if c.verified != c2.verified {
		t.Error("different verified")
	}
	if c.filename != c2.filename {
		t.Error("different filename")
	}
	if c.done != c2.done {
		t.Errorf("different done")
	}
	if c.size != c2.size {
		t.Errorf("different size")
	}
}
