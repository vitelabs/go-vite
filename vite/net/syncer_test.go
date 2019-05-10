package net

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/ledger"
)

func TestSplitChunks(t *testing.T) {
	type sample struct {
		from, to, size uint64
		cs             [][2]uint64
	}
	var samples = []sample{
		{1, 105, 30, [][2]uint64{{1, 30}, {31, 60}, {61, 90}, {91, 105}}},
		{1, 105, 200, [][2]uint64{{1, 105}}},
		{1, 1, 200, [][2]uint64{{1, 1}}},
	}

	for _, samp := range samples {
		cs := splitChunk(samp.from, samp.to, samp.size)
		if len(cs) != len(samp.cs) {
			t.Errorf("wrong split: %v", cs)
		} else {
			for i, c := range cs {
				if samp.cs[i] != c {
					t.Errorf("wrong chunk: %d - %d", c[0], c[1])
				}
			}
		}
	}
}

func TestHashHeightTree(t *testing.T) {
	hashHeightList1 := []*ledger.HashHeight{
		{100, mockHash()},
		{200, mockHash()},
		{300, mockHash()},
		{400, mockHash()},
	}
	hashHeightList2 := []*ledger.HashHeight{
		{100, mockHash()},
		{200, mockHash()},
		{300, mockHash()},
		{400, mockHash()},
		{500, mockHash()},
	}
	hashHeightList3 := make([]*ledger.HashHeight, 0, len(hashHeightList1)+1)
	for _, h := range hashHeightList1 {
		hashHeightList3 = append(hashHeightList3, h)
	}
	hashHeightList3 = append(hashHeightList3, &ledger.HashHeight{
		500, mockHash(),
	})

	tree := newHashHeightTree()
	tree.addBranch(hashHeightList1, nil)
	tree.addBranch(hashHeightList2, nil)
	tree.addBranch(hashHeightList3, nil)

	list := tree.bestBranch()

	// should be list3
	if len(hashHeightList3) != len(list) {
		t.Errorf("wrong length: %d", len(list))
	}
	for i, h := range list {
		if h.Hash != hashHeightList3[i].Hash || h.Height != hashHeightList3[i].Height {
			t.Errorf("wrong branch")
		}
	}
}

func TestConstructTasks(t *testing.T) {
	var hhs []*ledger.HashHeight

	const start uint64 = 100
	const end uint64 = 10000
	for i := start; i < end+1; i += 100 {
		hhs = append(hhs, &ledger.HashHeight{
			Height: i,
			Hash:   randomHash(),
		})
	}

	fmt.Printf("%d %d %d\n", start, end, len(hhs))

	point := &ledger.HashHeight{
		Height: start - 1,
		Hash:   randomHash(),
	}
	ts := constructTasks(point, hhs)

	fmt.Printf("%d tasks\n", len(ts))

	prevTask := &syncTask{
		to:      point.Height,
		endHash: point.Hash,
	}
	for _, tt := range ts {
		if tt.from != prevTask.to+1 || tt.prevHash != prevTask.endHash {
			t.Errorf("not continuous")
		}
		prevTask = tt
	}
}

func randomHash() (h types.Hash) {
	_, _ = rand.Read(h[:])
	return
}
