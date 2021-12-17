package abi

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/vitelabs/go-vite/v2/common/helper"
)

func TestNumberTypes(t *testing.T) {
	ubytes := make([]byte, helper.WordSize)
	ubytes[helper.WordSize-1] = 1

	unsigned := U256(big.NewInt(1))
	if !bytes.Equal(unsigned, ubytes) {
		t.Errorf("expected %x got %x", ubytes, unsigned)
	}
}
