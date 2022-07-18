package onroad_pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	val := orHeightValue(txs)

	tx, err := val.minTx()
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), *tx.FromIndex)
}
