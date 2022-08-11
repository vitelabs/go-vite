package onroad

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
)

type mochChain struct {
}

func (m mochChain) IsGenesisAccountBlock(block types.Hash) bool {
	return false
}
func Test_ExcludePairTrades(t *testing.T) {
	var blocks []*core.AccountBlock

	blocks = append(blocks, &core.AccountBlock{
		BlockType:      4,
		Hash:           types.DataHash([]byte{1}),
		Height:         5,
		AccountAddress: types.AddressAsset,
		ToAddress:      types.AddressQuota,
		FromBlockHash:  types.DataHash([]byte{2}),
		SendBlockList: []*core.AccountBlock{
			{
				BlockType: 2,
				Height:    0,
				Hash:      types.DataHash([]byte{3}),
				ToAddress: types.AddressQuota,
			},
			{
				BlockType: 2,
				Height:    0,
				Hash:      types.DataHash([]byte{4}),
				ToAddress: types.AddressAsset,
			},
		},
	})

	mm := ExcludePairTrades(&mochChain{}, blocks)

	assert.Equal(t, 2, len(mm))
	assert.Equal(t, 2, len(mm[types.AddressAsset]))
	assert.Equal(t, 1, len(mm[types.AddressQuota]))
	assert.Equal(t, types.DataHash([]byte{3}), mm[types.AddressQuota][0].Hash)
	assert.Equal(t, types.DataHash([]byte{4}), mm[types.AddressAsset][1].Hash)
	assert.Equal(t, types.DataHash([]byte{1}), mm[types.AddressAsset][0].Hash)

	for addr, blocks := range mm {
		for i, block := range blocks {
			t.Log(addr, i, block.AccountAddress, block.ToAddress, block.Hash, block.Height)
		}
	}
}
