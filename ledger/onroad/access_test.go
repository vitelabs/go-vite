package onroad

import (
	"testing"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/test_tools"
)

func Test_onroad(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)
	
	pageNum := 0
	addr, _ := types.HexToAddress("vite_0000000000000000000000000000000000000003f6af7459b9")
	for {
		blockList, err := c.GetOnRoadBlocksByAddr(addr, pageNum, 100)
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
