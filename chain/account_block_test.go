package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func GetAccountBlocksByHash(t *testing.T, chainInstance Chain, hashList []*types.Hash) {
	for _, hash := range hashList {
		block, err := chainInstance.GetAccountBlockByHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if block == nil || block.Hash != *hash {
			t.Fatal(fmt.Sprintf("block is error! block is %+v\n", block))
		}
	}
}

func GetAccountBlocksByHeight(t *testing.T, chainInstance Chain, addrList []*types.Address, heightList []uint64) {
	for index, height := range heightList {
		block, err := chainInstance.GetAccountBlockByHeight(addrList[index], height)
		if err != nil {
			t.Fatal(err)
		}
		if block.AccountAddress != *addrList[index] ||
			block.Height != height {
			t.Fatal("block is error!")
		}
	}
}

func IsAccountBlockExisted(t *testing.T, chainInstance Chain, hashList []*types.Hash) {
	for _, hash := range hashList {
		ok, err := chainInstance.IsAccountBlockExisted(hash)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal(fmt.Sprintf("error"))
		}
	}
}
