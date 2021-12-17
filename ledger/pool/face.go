package pool

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
)

type accBlocksSort []*core.AccountBlock

func (a accBlocksSort) Len() int {
	return len(a)
}
func (a accBlocksSort) Ids(i int) []string {
	var ids []string

	ids = append(ids, a[i].Hash.Hex())
	if blocks := a[i].SendBlockList; len(blocks) > 0 {
		for _, block := range blocks {
			ids = append(ids, block.Hash.Hex())
		}
	}
	return ids
}

func (a accBlocksSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a accBlocksSort) Inputs(i int) []string {
	var inputs []string
	block := a[i]
	if block.Height > types.GenesisHeight {
		inputs = append(inputs, a[i].PrevHash.Hex())
	}
	if block.IsReceiveBlock() && !block.IsGenesisBlock() {
		inputs = append(inputs, block.FromBlockHash.Hex())
	}
	return inputs
}
