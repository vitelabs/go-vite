package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
	"time"
)

func TestGetNeedSnapshotContent(t *testing.T) {
	chainInstance := getChainInstance()
	content := chainInstance.GetNeedSnapshotContent()
	for addr, item := range content {
		fmt.Printf("%s: %+v\n", addr.String(), item)
	}
}

func TestInsertSnapshotBlock(t *testing.T) {

}

func TestGetSnapshotBlocksByHash(t *testing.T) {
	chainInstance := getChainInstance()
	blocks, err := chainInstance.GetSnapshotBlocksByHash(nil, 100, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}

	blocks2, err2 := chainInstance.GetSnapshotBlocksByHash(nil, 100, true, true)
	if err2 != nil {
		t.Fatal(err2)
	}
	for index, block := range blocks2 {
		fmt.Printf("%d: %+v\n", index, block)
	}

	blocks3, err3 := chainInstance.GetSnapshotBlocksByHash(nil, 100, false, true)
	if err3 != nil {
		t.Fatal(err3)
	}
	for index, block := range blocks3 {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

func TestGetSnapshotBlocksByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	blocks, err := chainInstance.GetSnapshotBlocksByHeight(2, 10, false, false)
	if err != nil {
		t.Fatal(err)
	}
	for index, block := range blocks {
		fmt.Printf("%d: %+v\n", index, block)
	}
}

func TestGetSnapshotBlockByHeight(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetSnapshotBlockByHeight(1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	block2, err2 := chainInstance.GetSnapshotBlockByHeight(2)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", block2)
}

func TestGetSnapshotBlockByHash(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetSnapshotBlockByHash(&GenesisMintageSendBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	hash2, _ := types.HexToHash("f34e00c283f11728e28ccf2cf2138a7976b9ed7daaf7dbcc2ca598f66139f80d")
	block2, err2 := chainInstance.GetSnapshotBlockByHash(&hash2)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", block2)
}

func TestGetLatestSnapshotBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block := chainInstance.GetLatestSnapshotBlock()
	fmt.Printf("%+v\n", block)
}

func TestGetGenesisSnapshotBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block := chainInstance.GetGenesisSnapshotBlock()
	fmt.Printf("%+v\n", block)
}

func TestGetConfirmBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetConfirmBlock(&GenesisMintageSendBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	hash, _ := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	block2, err2 := chainInstance.GetConfirmBlock(&hash)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", block2)

	block3, err3 := chainInstance.GetConfirmBlock(&GenesisMintageBlock.Hash)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", block3)
}

func TestGetConfirmTimes(t *testing.T) {
	chainInstance := getChainInstance()
	times1, err := chainInstance.GetConfirmTimes(&GenesisMintageSendBlock.Hash)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", times1)

	hash, _ := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	times2, err2 := chainInstance.GetConfirmTimes(&hash)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", times2)

	times3, err3 := chainInstance.GetConfirmTimes(&GenesisMintageBlock.Hash)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", times3)
}

// TODO
func TestGetSnapshotBlockBeforeTime(t *testing.T) {
	chainInstance := getChainInstance()
	time1 := time.Now()
	block, err := chainInstance.GetSnapshotBlockBeforeTime(&time1)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	time2 := time.Unix(1535209021, 0)
	block2, err2 := chainInstance.GetSnapshotBlockBeforeTime(&time2)
	if err2 != nil {
		t.Fatal(err2)
	}
	fmt.Printf("%+v\n", block2)

	time3 := GenesisSnapshotBlock.Timestamp.Add(time.Second * 100)
	block3, err3 := chainInstance.GetSnapshotBlockBeforeTime(&time3)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", block3)
}

func TestGetConfirmAccountBlock(t *testing.T) {
	chainInstance := getChainInstance()
	block, err := chainInstance.GetConfirmAccountBlock(GenesisSnapshotBlock.Height, &GenesisMintageSendBlock.AccountAddress)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", block)

	block2, err2 := chainInstance.GetConfirmAccountBlock(GenesisSnapshotBlock.Height, &GenesisRegisterBlock.AccountAddress)
	if err2 != nil {
		t.Fatal(err2)
	}

	fmt.Printf("%+v\n", block2)

	block3, err3 := chainInstance.GetConfirmAccountBlock(GenesisSnapshotBlock.Height+10, &GenesisMintageSendBlock.AccountAddress)
	if err3 != nil {
		t.Fatal(err3)
	}
	fmt.Printf("%+v\n", block3)

	block4, err4 := chainInstance.GetConfirmAccountBlock(0, &GenesisMintageSendBlock.AccountAddress)
	if err4 != nil {
		t.Fatal(err4)
	}
	fmt.Printf("%+v\n", block4)
}

func TestDeleteSnapshotBlocksToHeight(t *testing.T) {

}
