package abi

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
	"testing"
)

func TestNumberTypes(t *testing.T) {
	ubytes := make([]byte, helper.WordSize)
	ubytes[helper.WordSize-1] = 1

	unsigned := U256(big.NewInt(1))
	if !bytes.Equal(unsigned, ubytes) {
		t.Errorf("expected %x got %x", ubytes, unsigned)
	}
}
