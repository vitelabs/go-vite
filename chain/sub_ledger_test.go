package chain

import (
	"fmt"
	"testing"
)

func TestGetSubLedgerByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	makeBlocks(chainInstance, 20000)
	chainInstance.Compressor().RunTask()
	files, dbRange := chainInstance.GetSubLedgerByHeight(100, 10000, true)
	for _, fileInfo := range files {
		fmt.Printf("%+v\n", fileInfo)
	}

	fmt.Printf("%+v\n", dbRange)
}

func TestGetSubLedgerByHash(t *testing.T) {
	chainInstance := getChainInstance()
	makeBlocks(chainInstance, 20000)
	chainInstance.Compressor().RunTask()
	files, dbRange, err := chainInstance.GetSubLedgerByHash(&GenesisSnapshotBlock.Hash, 1000, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, fileInfo := range files {
		fmt.Printf("%+v\n", fileInfo)
	}

	fmt.Printf("%+v\n", dbRange)
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
