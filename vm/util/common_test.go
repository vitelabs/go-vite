package util

import (
	"bytes"
	"math/big"
	"testing"
)

func TestToWorkSize(t *testing.T) {
	tests := []struct {
		byteSize uint64
		wordSize uint64
	}{
		{0, 0},
		{1, 1},
		{32, 1},
		{33, 2},
		{MaxUint64 - 31, MaxUint64 / 32},
		{MaxUint64 - 30, MaxUint64/32 + 1},
		{MaxUint64, MaxUint64/32 + 1},
	}
	for _, test := range tests {
		wordSize := ToWordSize(test.byteSize)
		if wordSize != test.wordSize {
			t.Fatalf("calculate word size from byte size fail, byte size: %v, expected %v, got %v", test.byteSize, test.wordSize, wordSize)
		}
	}
}

func TestBigUint64(t *testing.T) {
	tests := []struct {
		input  *big.Int
		result uint64
		ok     bool
	}{
		{new(big.Int).SetUint64(0), 0, false},
		{new(big.Int).SetUint64(MaxUint64), MaxUint64, false},
		{new(big.Int).SetUint64(MaxUint64).Add(new(big.Int).SetUint64(MaxUint64), Big1), 0, true},
		{new(big.Int).SetUint64(MaxUint64).Add(new(big.Int).SetUint64(MaxUint64), big.NewInt(2)), 1, true},
	}
	for _, test := range tests {
		result, ok := BigUint64(test.input)
		if result != test.result || ok != test.ok {
			t.Fatalf("get uint64 from big int fail: %v, expected [%v, %v], got [%v, %v]", test.input, test.result, test.ok, result, ok)
		}
	}
}

func TestRightPadBytes(t *testing.T) {
	tests := []struct {
		input  []byte
		len    int
		result []byte
	}{
		{[]byte{}, 8, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{[]byte{1, 2, 3}, 8, []byte{1, 2, 3, 0, 0, 0, 0, 0}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, 8, []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, 8, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
	}
	for _, test := range tests {
		result := RightPadBytes(test.input, test.len)
		if bytes.Compare(result, test.result) != 0 {
			t.Fatalf("right pad bytes fail, input: %v, len: %v, expected [%v], got [%v]", test.input, test.len, test.result, result)
		}
	}
}

func TestLeftPadBytes(t *testing.T) {
	tests := []struct {
		input  []byte
		len    int
		result []byte
	}{
		{[]byte{}, 8, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{[]byte{1, 2, 3}, 8, []byte{0, 0, 0, 0, 0, 1, 2, 3}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, 8, []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}, 8, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}},
	}
	for _, test := range tests {
		result := LeftPadBytes(test.input, test.len)
		if bytes.Compare(result, test.result) != 0 {
			t.Fatalf("left pad bytes fail, input: %v, len: %v, expected [%v], got [%v]", test.input, test.len, test.result, result)
		}
	}
}

func TestGetDataBig(t *testing.T) {
	tests := []struct {
		input  []byte
		start  *big.Int
		size   *big.Int
		result []byte
	}{
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, big.NewInt(0), big.NewInt(1), []byte{1}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, big.NewInt(0), big.NewInt(2), []byte{1, 2}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, big.NewInt(7), big.NewInt(1), []byte{8}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, big.NewInt(8), big.NewInt(1), []byte{0}},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 8}, big.NewInt(7), big.NewInt(2), []byte{8, 0}},
	}
	for _, test := range tests {
		result := GetDataBig(test.input, test.start, test.size)
		if bytes.Compare(result, test.result) != 0 {
			t.Fatalf("get data big fail, input: %v[%v, %v], expected [%v], got [%v]", test.input, test.start, test.size, test.result, result)
		}
	}
}

func TestBytesToString(t *testing.T) {
	tests := []struct {
		input  []byte
		result string
	}{
		{[]byte{116, 101, 115, 116}, "test"},
		{[]byte{116, 101, 0, 115, 116}, "te"},
		{[]byte{0, 116, 101, 115, 116}, ""},
		{[]byte{116, 101, 115, 116, 0, 0, 0, 0}, "test"},
	}
	for _, test := range tests {
		result := BytesToString(test.input)
		if result != test.result {
			t.Fatalf("get string from byte array fail, input: [%v], expected %v, got %v", test.input, test.result, result)
		}
	}
}
