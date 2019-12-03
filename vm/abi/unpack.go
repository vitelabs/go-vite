package abi

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"reflect"
)

// reads the integer based on its kind
func readInteger(kind reflect.Kind, b []byte) interface{} {
	switch kind {
	case reflect.Uint8:
		return b[len(b)-1]
	case reflect.Uint16:
		return binary.BigEndian.Uint16(b[len(b)-2:])
	case reflect.Uint32:
		return binary.BigEndian.Uint32(b[len(b)-4:])
	case reflect.Uint64:
		return binary.BigEndian.Uint64(b[len(b)-8:])
	case reflect.Int8:
		return int8(b[len(b)-1])
	case reflect.Int16:
		return int16(binary.BigEndian.Uint16(b[len(b)-2:]))
	case reflect.Int32:
		return int32(binary.BigEndian.Uint32(b[len(b)-4:]))
	case reflect.Int64:
		return int64(binary.BigEndian.Uint64(b[len(b)-8:]))
	default:
		return new(big.Int).SetBytes(b)
	}
}

// reads a bool
func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, errBadBool
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errBadBool
	}
}

// through reflection, creates a fixed array to be read from
func readFixedBytes(t Type, word []byte) (interface{}, error) {
	if t.T != FixedBytesTy {
		return nil, errInvalidlFixedBytesType
	}
	// convert
	array := reflect.New(t.Type).Elem()

	reflect.Copy(array, reflect.ValueOf(word[0:t.Size]))
	return array.Interface(), nil

}

func getFullElemSize(elem *Type) int {
	//all other should be counted as 32 (slices have pointers to respective elements)
	size := helper.WordSize
	//arrays wrap it, each element being the same size
	for elem.T == ArrayTy {
		size *= elem.Size
		elem = elem.Elem
	}
	return size
}

// iteratively unpack elements
func forEachUnpack(t Type, output []byte, start, size int) (interface{}, error) {
	if size < 0 {
		return nil, errNegativeInputSize(size)
	}
	if start+helper.WordSize*size > len(output) {
		return nil, errArrayOffsetOverflow(output, start, size)
	}

	// this value will become our slice or our array, depending on the type
	var refSlice reflect.Value

	if t.T == SliceTy {
		// declare our slice
		refSlice = reflect.MakeSlice(t.Type, size, size)
	} else if t.T == ArrayTy {
		// declare our array
		refSlice = reflect.New(t.Type).Elem()
	} else {
		return nil, errInvalidlArrayType
	}

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := helper.WordSize
	if t.T == ArrayTy {
		elemSize = getFullElemSize(t.Elem)
	}

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {

		inter, err := toGoType(i, *t.Elem, output)
		if err != nil {
			return nil, err
		}

		// append the item to our reflect slice
		refSlice.Index(j).Set(reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

// toGoType parses the output bytes and recursively assigns the value of these bytes
// into a go type with accordance with the ABI spec.
func toGoType(index int, t Type, output []byte) (interface{}, error) {
	if index+helper.WordSize > len(output) {
		return nil, errInsufficientLength(output, index)
	}

	var (
		returnOutput []byte
		begin, end   int
		err          error
	)

	// if we require a length prefix, find the beginning word and size returned.
	if t.requiresLengthPrefix() {
		begin, end, err = lengthPrefixPointsTo(index, output)
		if err != nil {
			return nil, err
		}
	} else {
		returnOutput = output[index : index+helper.WordSize]
	}

	switch t.T {
	case SliceTy:
		return forEachUnpack(t, output, begin, end)
	case ArrayTy:
		return forEachUnpack(t, output, index, t.Size)
	case StringTy: // variable arrays are written at the end of the return bytes
		return string(output[begin : begin+end]), nil
	case IntTy, UintTy:
		return readInteger(t.Kind, returnOutput), nil
	case BoolTy:
		return readBool(returnOutput)
	case AddressTy:
		addr, _ := types.BytesToAddress(returnOutput[helper.WordSize-types.AddressSize : helper.WordSize])
		return addr, nil
	case GidTy:
		gid, _ := types.BytesToGid(returnOutput[helper.WordSize-types.GidSize : helper.WordSize])
		return gid, nil
	case TokenIdTy:
		tokenId, _ := types.BytesToTokenTypeId(returnOutput[helper.WordSize-types.TokenTypeIdSize : helper.WordSize])
		return tokenId, nil
	case BytesTy:
		return output[begin : begin+end], nil
	case FixedBytesTy:
		return readFixedBytes(t, returnOutput)
	default:
		return nil, errUnknownType(t)
	}
}

// interprets a 32 byte slice as an offset and then determines which indice to look to decode the type.
func lengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
	intpool := util.PoolOfIntPools.Get()
	defer util.PoolOfIntPools.Put(intpool)
	bigOffsetEnd := intpool.Get().SetBytes(output[index : index+helper.WordSize])
	defer intpool.Put(bigOffsetEnd)
	bigOffsetEnd.Add(bigOffsetEnd, helper.Big32)
	outputLength := intpool.Get().SetInt64(int64(len(output)))
	defer intpool.Put(outputLength)

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, errBigSliceOffsetOverflow(bigOffsetEnd, outputLength)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, errBigOffsetOverflow(bigOffsetEnd)
	}

	offsetEnd := int(bigOffsetEnd.Uint64())
	lengthBig := intpool.Get().SetBytes(output[offsetEnd-helper.WordSize : offsetEnd])
	defer intpool.Put(lengthBig)

	totalSize := intpool.GetZero()
	defer intpool.Put(totalSize)
	totalSize.Add(totalSize, bigOffsetEnd)
	totalSize.Add(totalSize, lengthBig)
	if totalSize.BitLen() > 63 {
		return 0, 0, errBigLengthOverflow(totalSize)
	}

	if totalSize.Cmp(outputLength) > 0 {
		return 0, 0, errInsufficientBigLength(outputLength, totalSize)
	}
	start = int(bigOffsetEnd.Uint64())
	length = int(lengthBig.Uint64())
	return
}
