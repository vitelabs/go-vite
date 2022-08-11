package onroad_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/types"
)

func Test_orHeightValue(t *testing.T) {
	i0 := uint32(0)
	i1 := uint32(1)
	i2 := uint32(2)
	txs := []OnroadTx{
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 2,
			FromHash:   [32]byte{2},
			FromIndex:  &i2,
		},
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 0,
			FromHash:   [32]byte{0},
			FromIndex:  &i0,
		},
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 0,
			FromHash:   [32]byte{1},
			FromIndex:  &i1,
		},
	}

	val, err := newOrHeightValueFromOnroadTxs(txs)
	assert.NoError(t, err)

	tx, err := val.minTx()
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), *tx.FromIndex)
}

func Test_orHeightValue_dirtyTxs(t *testing.T) {
	i0 := uint32(0)
	i1 := uint32(1)
	i2 := uint32(2)
	txs := []OnroadTx{
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 2,
			FromHash:   [32]byte{2},
			FromIndex:  &i2,
		},
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 0,
			FromHash:   [32]byte{0},
			FromIndex:  &i0,
		},
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 0,
			FromHash:   types.DataHash([]byte{3}),
			FromIndex:  nil,
		},
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 0,
			FromHash:   [32]byte{1},
			FromIndex:  &i1,
		},
		{
			FromAddr:   [21]byte{},
			ToAddr:     [21]byte{},
			FromHeight: 0,
			FromHash:   types.DataHash([]byte{4}),
			FromIndex:  nil,
		},
	}

	val, err := newOrHeightValueFromOnroadTxs(txs)
	assert.NoError(t, err)

	dirtyTxs := val.dirtyTxs()
	// for _, tx := range dirtyTxs {
	// 	t.Log(tx.String())
	// }
	assert.Equal(t, 2, len(dirtyTxs))
	assert.Equal(t, types.DataHash([]byte{3}), dirtyTxs[0].FromHash)
	assert.Equal(t, types.DataHash([]byte{4}), dirtyTxs[1].FromHash)

	for i, sub := range dirtyTxs {
		j := uint32(i)
		sub.FromIndex = &j
	}
	{
		dirtyTxs := val.dirtyTxs()
		// for _, tx := range dirtyTxs {
		// 	t.Log(tx.String())
		// }
		assert.Equal(t, 0, len(dirtyTxs))
	}
}
