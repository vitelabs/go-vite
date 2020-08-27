package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func Test_onroad(t *testing.T) {
	chainInstance, err := NewChainInstance(unitTestPath, false)
	if err != nil {
		t.Fatal(err)
	}
	pageNum := 0
	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000003f6af7459b9")
	for {
		blockList, err := chainInstance.GetOnRoadBlocksByAddr(addr, pageNum, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(blockList) <= 0 {
			break
		}
		t.Log(len(blockList))
		for _, block := range blockList {
			t.Log("hash", block.Hash, "height", block.Height)
		}
		pageNum++
	}
}
