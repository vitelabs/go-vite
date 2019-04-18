package net

import (
	"encoding/json"
	"fmt"

	"github.com/vitelabs/go-vite/interfaces"
)

func ExampleSyncDetail() {
	var detail = SyncDetail{
		From:    10,
		To:      100,
		Current: 20,
		State:   Syncing,
		Cache: interfaces.SegmentList{
			{11, 30},
			{31, 40},
			{41, 50},
		},
	}

	data, err := json.Marshal(detail)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", data)
	// Output:
	// {"from":10,"to":100,"current":20,"state":1,"downloader":{"tasks":null,"connections":null},"cache":[[11,30],[31,40],[41,50]]}
}
