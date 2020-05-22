package client

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestSBPUpgrade(t *testing.T) {

	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	sbpMap := make(map[types.Address]string)
	versionMap := make(map[string]uint32)

	sbpList, err := rpc.GetSBPVoteList()
	for _, sbp := range sbpList {
		fmt.Println(sbp.Name, sbp.BlockProducingAddress)
		sbpMap[sbp.BlockProducingAddress] = sbp.Name
	}

	latestHeight, err := strconv.ParseUint(rpc.GetSnapshotChainHeight(), 10, 64)

	if err != nil {
		panic(err)
	}
	blocks, err := rpc.GetSnapshotBlocks(latestHeight, 500)
	if err != nil {
		panic(err)
	}
	for _, block := range blocks {
		producer := block.Producer
		version := block.Version

		fmt.Println(sbpMap[producer], producer, version)
		versionMap[sbpMap[producer]] = version
	}

	for k, version := range versionMap {
		if version < 8 {
			fmt.Println(k, version)
		}
	}

	for k, version := range versionMap {
		if version >= 8 {
			fmt.Print(k, ",")
		}
	}
	fmt.Println()
}
