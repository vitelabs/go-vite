package abi

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestPack(t *testing.T) {
	for i, test := range []struct {
		typ string

		input  interface{}
		output []byte
	}{
		{
			"uint8",
			uint8(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint8[]",
			[]uint8{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint16",
			uint16(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint16[]",
			[]uint16{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint32",
			uint32(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint32[]",
			[]uint32{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint64",
			uint64(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint64[]",
			[]uint64{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint256",
			big.NewInt(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"uint256[]",
			[]*big.Int{big.NewInt(1), big.NewInt(2)},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int8",
			int8(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int8[]",
			[]int8{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int16",
			int16(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int16[]",
			[]int16{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int32",
			int32(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int32[]",
			[]int32{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int64",
			int64(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int64[]",
			[]int64{1, 2},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int256",
			big.NewInt(2),
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"int256[]",
			[]*big.Int{big.NewInt(1), big.NewInt(2)},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			"bytes1",
			[1]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes2",
			[2]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes3",
			[3]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes4",
			[4]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes5",
			[5]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes6",
			[6]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes7",
			[7]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes8",
			[8]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes9",
			[9]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes10",
			[10]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes11",
			[11]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes12",
			[12]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes13",
			[13]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes14",
			[14]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes15",
			[15]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes16",
			[16]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes17",
			[17]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes18",
			[18]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes19",
			[19]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes20",
			[20]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes21",
			[21]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes22",
			[22]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes23",
			[23]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes24",
			[24]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes24",
			[24]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes25",
			[25]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes26",
			[26]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes27",
			[27]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes28",
			[28]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes29",
			[29]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes30",
			[30]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes31",
			[31]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"bytes32",
			[32]byte{1},
			helper.HexToBytes("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"address[]",
			[]types.Address{{1}, {2}},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000"),
		},
		{
			"gid[]",
			[]types.Gid{{1}, {2}},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002000000000000000000"),
		},
		{
			"tokenId[]",
			[]types.TokenTypeId{{1}, {2}},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000002000000000000000000"),
		},
		{
			"bytes32[]",
			[]types.Hash{{1}, {2}},
			helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000000201000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			"string",
			"foobar",
			helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000006666f6f6261720000000000000000000000000000000000000000000000000000"),
		},
		{
			"string",
			"0x02",
			helper.HexToBytes("00000000000000000000000000000000000000000000000000000000000000043078303200000000000000000000000000000000000000000000000000000000"),
		},
	} {
		typ, err := NewType(test.typ)
		if err != nil {
			t.Fatalf("%v failed. Unexpected parse error: %v", i, err)
		}

		output, err := typ.pack(reflect.ValueOf(test.input))
		if err != nil {
			t.Fatalf("%v failed. Unexpected pack error: %v", i, err)
		}

		if !bytes.Equal(output, test.output) {
			t.Errorf("%d failed. Expected bytes: '%x' Got: '%x'", i, test.output, output)
		}
	}
}

func TestMethodPack(t *testing.T) {
	abi, err := JSONToABIContract(strings.NewReader(jsondata2))
	if err != nil {
		t.Fatal(err)
	}

	sig := abi.Methods["slice"].Id()
	sig = append(sig, helper.LeftPadBytes([]byte{1}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes([]byte{2}, helper.WordSize)...)

	packed, err := abi.PackMethod("slice", []uint32{1, 2})
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}

	var addrA, addrB = types.Address{1}, types.Address{2}
	sig = abi.Methods["sliceAddress"].Id()
	sig = append(sig, helper.LeftPadBytes([]byte{32}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes([]byte{2}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes(addrA[:], helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes(addrB[:], helper.WordSize)...)

	packed, err = abi.PackMethod("sliceAddress", []types.Address{addrA, addrB})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}

	var addrC, addrD = types.Address{3}, types.Address{4}
	sig = abi.Methods["sliceMultiAddress"].Id()
	sig = append(sig, helper.LeftPadBytes([]byte{64}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes([]byte{160}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes([]byte{2}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes(addrA[:], helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes(addrB[:], helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes([]byte{2}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes(addrC[:], helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes(addrD[:], helper.WordSize)...)

	packed, err = abi.PackMethod("sliceMultiAddress", []types.Address{addrA, addrB}, []types.Address{addrC, addrD})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}

	sig = abi.Methods["slice256"].Id()
	sig = append(sig, helper.LeftPadBytes([]byte{1}, helper.WordSize)...)
	sig = append(sig, helper.LeftPadBytes([]byte{2}, helper.WordSize)...)

	packed, err = abi.PackMethod("slice256", []*big.Int{big.NewInt(1), big.NewInt(2)})
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}
}

func TestPackNumber(t *testing.T) {
	tests := []struct {
		value  reflect.Value
		packed []byte
	}{
		// Protocol limits
		{reflect.ValueOf(0), helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000000")},
		{reflect.ValueOf(1), helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000000001")},
		{reflect.ValueOf(-1), helper.HexToBytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")},

		// Type corner cases
		{reflect.ValueOf(uint8(math.MaxUint8)), helper.HexToBytes("00000000000000000000000000000000000000000000000000000000000000ff")},
		{reflect.ValueOf(uint16(math.MaxUint16)), helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000ffff")},
		{reflect.ValueOf(uint32(math.MaxUint32)), helper.HexToBytes("00000000000000000000000000000000000000000000000000000000ffffffff")},
		{reflect.ValueOf(uint64(math.MaxUint64)), helper.HexToBytes("000000000000000000000000000000000000000000000000ffffffffffffffff")},

		{reflect.ValueOf(int8(math.MaxInt8)), helper.HexToBytes("000000000000000000000000000000000000000000000000000000000000007f")},
		{reflect.ValueOf(int16(math.MaxInt16)), helper.HexToBytes("0000000000000000000000000000000000000000000000000000000000007fff")},
		{reflect.ValueOf(int32(math.MaxInt32)), helper.HexToBytes("000000000000000000000000000000000000000000000000000000007fffffff")},
		{reflect.ValueOf(int64(math.MaxInt64)), helper.HexToBytes("0000000000000000000000000000000000000000000000007fffffffffffffff")},

		{reflect.ValueOf(int8(math.MinInt8)), helper.HexToBytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80")},
		{reflect.ValueOf(int16(math.MinInt16)), helper.HexToBytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8000")},
		{reflect.ValueOf(int32(math.MinInt32)), helper.HexToBytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffff80000000")},
		{reflect.ValueOf(int64(math.MinInt64)), helper.HexToBytes("ffffffffffffffffffffffffffffffffffffffffffffffff8000000000000000")},
	}
	for i, tt := range tests {
		packed, err := packNum(tt.value)
		if err != nil {
			t.Errorf("test %d: pack failed, want %x, err %v", i, tt.packed, err)
		}
		if !bytes.Equal(packed, tt.packed) {
			t.Errorf("test %d: pack mismatch: have %x, want %x", i, packed, tt.packed)
		}
	}
}
