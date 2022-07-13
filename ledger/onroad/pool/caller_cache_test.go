package onroad_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/types"
)

func Test_callerCache_1(t *testing.T) {
	cases := generateCases()

	for _, item := range cases {
		cc := NewCallerCache()
		for _, tx := range item.txs {
			err := cc.addTx(&item.caller, tx, true)
			assert.NoError(t, err)
		}
		{
			ohv, err := cc.getFrontTxByCaller(&item.caller)
			assert.NoError(t, err)
			assert.NotNil(t, ohv)

			assert.Equal(t, item.expectedFrontTxHeight, ohv.Height)
			assert.Equal(t, item.expectedFrontTxSize, len(ohv.Hashes))
		}
		{
			ohvs, err := cc.getFrontTxOfAllCallers()
			assert.NoError(t, err)
			assert.NotNil(t, ohvs)
			assert.Equal(t, 1, len(ohvs))

			assert.Equal(t, item.expectedFrontTxHeight, ohvs[0].Height)
			assert.Equal(t, item.expectedFrontTxSize, len(ohvs[0].Hashes))
		}
	}
}

type normalCase struct {
	txs    []orHashHeight
	caller types.Address

	expectedFrontTxHeight uint64
	expectedFrontTxSize   int
}

func generateCases() []normalCase {
	return []normalCase{{
		txs: []orHashHeight{
			{
				Hash:     types.DataHash([]byte{1}),
				Height:   2,
				SubIndex: nil,
			},
			{
				Hash:     types.DataHash([]byte{2}),
				Height:   1,
				SubIndex: nil,
			},
			{
				Hash:     types.DataHash([]byte{3}),
				Height:   1,
				SubIndex: nil,
			},
		},
		caller:                types.AddressAsset,
		expectedFrontTxHeight: 1,
		expectedFrontTxSize:   2,
	}, {
		txs: []orHashHeight{
			{
				Hash:     types.DataHash([]byte{1}),
				Height:   1,
				SubIndex: nil,
			},
			{
				Hash:     types.DataHash([]byte{2}),
				Height:   1,
				SubIndex: nil,
			},
			{
				Hash:     types.DataHash([]byte{3}),
				Height:   1,
				SubIndex: nil,
			},
		},
		caller:                types.AddressAsset,
		expectedFrontTxHeight: 1,
		expectedFrontTxSize:   3,
	},
		{
			txs: []orHashHeight{
				{
					Hash:     types.DataHash([]byte{1}),
					Height:   4,
					SubIndex: nil,
				},
				{
					Hash:     types.DataHash([]byte{2}),
					Height:   1,
					SubIndex: nil,
				},
				{
					Hash:     types.DataHash([]byte{3}),
					Height:   3,
					SubIndex: nil,
				},
				{
					Hash:     types.DataHash([]byte{4}),
					Height:   1,
					SubIndex: nil,
				},
			},
			caller:                types.AddressAsset,
			expectedFrontTxHeight: 1,
			expectedFrontTxSize:   2,
		},
	}
}
