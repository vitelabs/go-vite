package net

import (
	"encoding/json"
	"fmt"
	"testing"

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

func ExampleSyncDetail() {
	var detail = SyncDetail{
		From:    10,
		To:      100,
		Current: 20,
		State:   Syncing,
		Cache: interfaces.SegmentList{
			{11, 30},
			{31, 40},
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
