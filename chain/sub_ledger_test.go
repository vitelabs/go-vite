package chain

import (
	"fmt"
	"testing"
)

func TestGetSubLedgerByHeight(t *testing.T) {

}

func TestGetSubLedgerByHash(t *testing.T) {

}

func TestGetConfirmSubLedger(t *testing.T) {
	chainInstance := getChainInstance()
	snapshotBlocks, subLedger, err := chainInstance.GetConfirmSubLedger(0, 100)
	if err != nil {
		t.Fatal(err)
	}

	for index, block := range snapshotBlocks {
		fmt.Printf("%d: %+v\n", index, block)
	}

	fmt.Println()

	for addr, blocks := range subLedger {
		fmt.Printf("%s\n", addr.String())
		for index, block := range blocks {
			fmt.Printf("%d: %+v\n", index, block)
		}
	}

}
