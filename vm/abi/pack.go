package abi

import (
	"github.com/vitelabs/go-vite/common/helper"
	"math/big"
	"reflect"
)

// packBytesSlice packs the given bytes as [L, V] as the canonical representation
// bytes slice
func packBytesSlice(bytes []byte, l int) []byte {
	len := packNum(reflect.ValueOf(l))
	return append(len, helper.RightPadBytes(bytes, (l+helper.WordSize-1)/helper.WordSize*helper.WordSize)...)
}

// packElement packs the given reflect value according to the abi specification in
// t.
func packElement(t Type, reflectValue reflect.Value) []byte {
	switch t.T {
	case IntTy, UintTy:
		return packNum(reflectValue)
	case StringTy:
		return packBytesSlice([]byte(reflectValue.String()), reflectValue.Len())
	case AddressTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}

		return helper.LeftPadBytes(reflectValue.Bytes(), helper.WordSize)
	case GidTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}

		return helper.LeftPadBytes(reflectValue.Bytes(), helper.WordSize)
	case TokenIdTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}

		return helper.LeftPadBytes(reflectValue.Bytes(), helper.WordSize)
	case BoolTy:
		if reflectValue.Bool() {
			return helper.PaddedBigBytes(helper.Big1, helper.WordSize)
		}
		return helper.PaddedBigBytes(helper.Big0, helper.WordSize)
	case BytesTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}
		return packBytesSlice(reflectValue.Bytes(), reflectValue.Len())
	case FixedBytesTy, FunctionTy:
		if reflectValue.Kind() == reflect.Array {
			reflectValue = mustArrayToByteSlice(reflectValue)
		}
		return helper.RightPadBytes(reflectValue.Bytes(), helper.WordSize)
	default:
		panic("abi: fatal error")
	}
}

// packNum packs the given number (using the reflect value) and will cast it to appropriate number representation
func packNum(value reflect.Value) []byte {
	switch kind := value.Kind(); kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return U256(new(big.Int).SetUint64(value.Uint()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return U256(big.NewInt(value.Int()))
	case reflect.Ptr:
		return U256(value.Interface().(*big.Int))
	default:
		panic("abi: fatal error")
	}

}
