package abi

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"reflect"
)

var (
	bigT      = reflect.TypeOf(&big.Int{})
	derefbigT = reflect.TypeOf(big.Int{})
	uint8T    = reflect.TypeOf(uint8(0))
	uint16T   = reflect.TypeOf(uint16(0))
	uint32T   = reflect.TypeOf(uint32(0))
	uint64T   = reflect.TypeOf(uint64(0))
	int8T     = reflect.TypeOf(int8(0))
	int16T    = reflect.TypeOf(int16(0))
	int32T    = reflect.TypeOf(int32(0))
	int64T    = reflect.TypeOf(int64(0))
	addressT  = reflect.TypeOf(types.Address{})
	gidT      = reflect.TypeOf(types.Gid{})
	tokenIdT  = reflect.TypeOf(types.TokenTypeId{})
)

// U256 converts chain big Int into chain 256bit VM number.
func U256(n *big.Int) []byte {
	return helper.PaddedBigBytes(helper.U256(n), helper.WordSize)
}
