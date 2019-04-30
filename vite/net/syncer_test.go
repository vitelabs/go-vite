package net

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/interfaces"
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
	tree.addBranch(hashHeightList1)
	tree.addBranch(hashHeightList2)
	tree.addBranch(hashHeightList3)

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

func ExampleSyncDetail() {
	var detail = SyncDetail{
		From:    10,
		To:      100,
		Current: 20,
		State:   Syncing,
		Cache: interfaces.SegmentList{
			{[2]uint64{11, 30}, types.Hash{}, types.Hash{}},
			{[2]uint64{31, 40}, types.Hash{}, types.Hash{}},
		},
	}

	data, err := json.Marshal(detail)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", data)
	// Output:
	// {"from":10,"to":100,"current":20,"state":"Synchronising","downloader":{"tasks":null,"connections":null},"cache":[[11,30],[31,40]]}
}
