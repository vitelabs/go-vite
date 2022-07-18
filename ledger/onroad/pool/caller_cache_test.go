package onroad_pool

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/v2/common/types"
)

func Test_callerCache_1(t *testing.T) {
	cases := generateCases()
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)
	dir := path.Join(homeDir, ".gvite", "tmp", "onroad")
	for _, item := range cases {
		os.RemoveAll(dir)
		d, err := leveldb.OpenFile(dir, nil)
		if err != nil {
			panic(err)
		}
		cc := NewCallerCache(types.AddressDexFund, newOnroadStorage(d))
		for _, tx := range item.txs {
			err := cc.addTx(&item.caller, tx, true)
			assert.NoError(t, err)
		}
		{
			ohv, err := cc.getFrontTxByCaller(&item.caller)
			assert.NoError(t, err)
			assert.NotNil(t, ohv)

			txs := *ohv
			assert.Equal(t, item.expectedFrontTxHeight, txs[0].FromHeight)
			assert.Equal(t, item.expectedFrontTxSize, len(*ohv))
		}
		{
			ohvs, err := cc.getFrontTxOfAllCallers()
			assert.NoError(t, err)
			assert.NotNil(t, ohvs)
			assert.Equal(t, 1, len(ohvs))

			txs := *ohvs[0]
			assert.Equal(t, item.expectedFrontTxHeight, txs[0].FromHeight)
			assert.Equal(t, item.expectedFrontTxSize, len(*ohvs[0]))
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

// ________________________________________________________________________

func Test_callerCache_2(t *testing.T) {

	type orHashHeightAction struct {
		orHashHeight
		newOrDestory     bool // true -> new, false -> destory>
		insertOrRollback bool // true -> insert, false -> rollback
	}

	type caseStruct struct {
		actions []orHashHeightAction
		caller  types.Address

		expectedFrontTxHeight uint64
		expectedFrontTxSize   int
	}

	defaultCaller := types.AddressAsset

	cases := []caseStruct{
		{
			actions: []orHashHeightAction{
				{
					orHashHeight: orHashHeight{
						Hash:     types.DataHash([]byte{1}),
						Height:   1,
						SubIndex: new(uint32),
					},
					newOrDestory:     true,
					insertOrRollback: true,
				},
				{
					orHashHeight: orHashHeight{
						Hash:     types.DataHash([]byte{2}),
						Height:   2,
						SubIndex: new(uint32),
					},
					newOrDestory:     true,
					insertOrRollback: true,
				},
				{
					orHashHeight: orHashHeight{
						Hash:     types.DataHash([]byte{3}),
						Height:   1,
						SubIndex: new(uint32),
					},
					newOrDestory:     true,
					insertOrRollback: true,
				},
				{
					orHashHeight: orHashHeight{
						Hash:     types.DataHash([]byte{2}),
						Height:   2,
						SubIndex: new(uint32),
					},
					newOrDestory:     false,
					insertOrRollback: true,
				},
				{
					orHashHeight: orHashHeight{
						Hash:     types.DataHash([]byte{3}),
						Height:   1,
						SubIndex: new(uint32),
					},
					newOrDestory:     false,
					insertOrRollback: true,
				},
			},
			caller:                defaultCaller,
			expectedFrontTxHeight: 1,
			expectedFrontTxSize:   1,
		},
	}

	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)
	dir := path.Join(homeDir, ".gvite", "tmp", "onroad")

	for _, item := range cases {
		os.RemoveAll(dir)
		d, err := leveldb.OpenFile(dir, nil)
		if err != nil {
			panic(err)
		}
		cc := NewCallerCache(types.AddressGovernance, newOnroadStorage(d))

		for _, action := range item.actions {
			if action.newOrDestory {
				err := cc.addTx(&item.caller, action.orHashHeight, true)
				assert.NoError(t, err)
			} else {
				err := cc.rmTx(&item.caller, false, action.orHashHeight, false)
				assert.NoError(t, err)
			}
		}
		{
			ohv, err := cc.getFrontTxByCaller(&item.caller)
			assert.NoError(t, err)
			assert.NotNil(t, ohv)

			txs := *ohv
			assert.Equal(t, item.expectedFrontTxHeight, txs[0].FromHeight)
			assert.Equal(t, item.expectedFrontTxSize, len(*ohv))
		}
		{
			ohvs, err := cc.getFrontTxOfAllCallers()
			assert.NoError(t, err)
			assert.NotNil(t, ohvs)
			assert.Equal(t, 1, len(ohvs))

			txs := *ohvs[0]
			assert.Equal(t, item.expectedFrontTxHeight, txs[0].FromHeight)
			assert.Equal(t, item.expectedFrontTxSize, len(*ohvs[0]))
		}
	}

}
