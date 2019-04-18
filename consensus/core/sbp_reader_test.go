package core

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestBigInt(t *testing.T) {
	n := big.NewInt(10)

	bt := BigInt{Int: *n}

	bt2 := bt.SetInt64(11)

	t.Log(bt.String(), bt2.String())

	assert.Equal(t, bt.String(), "11")
	assert.Equal(t, bt2.String(), "11")
}
