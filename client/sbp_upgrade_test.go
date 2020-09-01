package client

import (
	"fmt"
	"strconv"
	"testing"

	"gotest.tools/assert"

	"github.com/vitelabs/go-vite/common/types"
)

func TestSBPUpgrade(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	assert.NilError(t, err)

	sbpMap := make(map[types.Address]string)
	versionMap := make(map[string]uint32)

	sbpList, err := rpc.GetSBPVoteList()
	assert.NilError(t, err)

	for _, sbp := range sbpList {
		fmt.Println(sbp.Name, sbp.BlockProducingAddress)
		sbpMap[sbp.BlockProducingAddress] = sbp.Name
	}

	latestHeight, err := strconv.ParseUint(rpc.GetSnapshotChainHeight(), 10, 64)
	assert.NilError(t, err)

	blocks, err := rpc.GetSnapshotBlocks(latestHeight, 500)
	assert.NilError(t, err)

	for _, block := range blocks {
		producer := block.Producer
		version := block.Version

		fmt.Println(sbpMap[producer], producer, version)
		versionMap[sbpMap[producer]] = version
	}

	_version := uint32(9)
	for k, version := range versionMap {
		if version < _version {
			fmt.Println(k, version)
		}
	}

	for k, version := range versionMap {
		if version >= _version {
			fmt.Print(k, ",")
		}
	}
	fmt.Println()

	for k, version := range versionMap {
		if version < _version {
			fmt.Print(k, ",")
		}
	}
	fmt.Println()
}
