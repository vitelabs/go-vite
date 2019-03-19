package chain_unittest

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain_db/access"
	"math/big"
	"testing"
)

func getEventTypeDescription(eventType byte) string {
	desc := ""
	switch eventType {
	case access.AddAccountBlocksEvent:
		desc = "AddAccountBlocksEvent"
	case access.DeleteAccountBlocksEvent:
		desc = "DeleteAccountBlocksEvent"
	case access.AddSnapshotBlocksEvent:
		desc = "AddSnapshotBlocksEvent"
	case access.DeleteSnapshotBlocksEvent:
		desc = "DeleteSnapshotBlocksEvent"
	}
	return desc
}

func TestChain_GetEvent(t *testing.T) {
	const PRINT_PER_COUNT = 100000
	chainInstance := NewChainInstance("testdata", false)
	latestBlockEventId, _ := chainInstance.GetLatestBlockEventId()
	fmt.Println("Start getEvent() test")

	count := 0
	result := make(map[byte]uint64)
	for i := latestBlockEventId; i > 0; i-- {
		count++
		types, _, _ := chainInstance.GetEvent(i)
		result[types]++

		if count%PRINT_PER_COUNT == 0 {
			fmt.Printf("Has tested %d/%d\n", count, latestBlockEventId)
		}
	}

	fmt.Printf("Has tested %d/%d\n", count, latestBlockEventId)
	fmt.Println("Complete getEvent() test ")

	fmt.Printf("======Test result======")
	for eventType, eventCount := range result {
		fmt.Printf("%s: %d times\n", getEventTypeDescription(eventType), eventCount)
	}
	fmt.Printf("======Test result======")

}

func TestDifficulty(t *testing.T) {
	new(big.Int).SetString("", 10)
}
