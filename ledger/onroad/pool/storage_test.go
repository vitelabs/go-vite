package onroad_pool

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/v2/common/types"
)

func newTestStorage(t *testing.T) *onroadStorage {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)
	dir := path.Join(homeDir, ".gvite", "tmp", "onroad")
	d, err := leveldb.OpenFile(dir, nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	return newOnroadStorage(d)
}

func clearTestStorage(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)
	dir := path.Join(homeDir, ".gvite", "tmp", "onroad")
	os.RemoveAll(dir)
}

func TestOnroadStorages(t *testing.T) {

	type txAction struct {
		fromHash   byte
		fromHeight uint64
		action     bool
		index      int64
	}

	type storageCase struct {
		actions     []txAction
		expectedTxs []txAction
	}

	fromAddr := types.AddressDexFund
	toAddr := types.AddressDexTrade
	cases := []storageCase{
		{
			actions: []txAction{
				{
					fromHash:   1,
					fromHeight: 10,
					action:     true,
					index:      -1,
				},
				{
					fromHash:   2,
					fromHeight: 10,
					action:     true,
					index:      0,
				},
				{
					fromHash:   3,
					fromHeight: 8,
					action:     true,
					index:      0,
				}, {
					fromHash:   3,
					fromHeight: 8,
					action:     false,
					index:      0,
				},
			},
			expectedTxs: []txAction{
				{
					fromHash:   2,
					fromHeight: 10,
					index:      0,
				},
				{
					fromHash:   1,
					fromHeight: 10,
					index:      -1,
				},
			},
		},
		{
			actions: []txAction{
				{
					fromHash:   1,
					fromHeight: 10,
					action:     true,
					index:      -1,
				},
				{
					fromHash:   2,
					fromHeight: 10,
					action:     true,
					index:      0,
				},
				{
					fromHash:   3,
					fromHeight: 8,
					action:     true,
					index:      1,
				}, {
					fromHash:   2,
					fromHeight: 10,
					action:     false,
					index:      0,
				},
			},
			expectedTxs: []txAction{
				{
					fromHash:   3,
					fromHeight: 8,
					index:      1,
				},
			},
		},
	}

	clearTestStorage(t)
	for _, ce := range cases {
		storage := newTestStorage(t)
		for _, action := range ce.actions {
			var index *uint32
			if action.index >= 0 {
				dd := uint32(action.index)
				index = &dd
			}

			onroadTx := OnroadTx{
				FromAddr:   fromAddr,
				ToAddr:     toAddr,
				FromHeight: action.fromHeight,
				FromHash:   types.DataHash([]byte{action.fromHash}),
				FromIndex:  index,
			}
			if action.action {
				storage.insertOnRoadTx(onroadTx)
			} else {
				storage.deleteOnRoadTx(onroadTx)
			}
		}
		ot, err := storage.getFirstOnroadTx(toAddr, fromAddr)

		assert.NoError(t, err)

		for i, tx := range ot {
			assert.Equal(t, ce.expectedTxs[i].fromHeight, tx.FromHeight)
			assert.Equal(t, types.DataHash([]byte{ce.expectedTxs[i].fromHash}), tx.FromHash)
			if tx.FromIndex == nil {
				assert.Equal(t, int64(-1), ce.expectedTxs[i].index)
			} else {
				assert.Equal(t, ce.expectedTxs[i].index, int64(*tx.FromIndex))
			}
		}
		storage.db.Close()
		clearTestStorage(t)
	}

}
