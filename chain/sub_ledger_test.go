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
	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	fmt.Println(latestSnapshotBlock)

	//makeBlocks(chainInstance, 1000)
	snapshotBlocks, subLedger, err := chainInstance.GetConfirmSubLedger(0, 2000)
	if err != nil {
		t.Fatal(err)
	}

	count := len(snapshotBlocks)
	addrCount := 0
	for _, blocks := range subLedger {
		addrCount++
		count += len(blocks)

	}
	fmt.Println(addrCount)

	fmt.Println(count)
}
