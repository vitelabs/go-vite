package vm

import (
	"bytes"
	"math/big"
	"testing"
)

func TestMin(t *testing.T) {
	tests := []struct {
		x, y, result uint64
	}{
		{1, 2, 1},
		{1, 1, 1},
		{2, 1, 1},
	}
	for _, test := range tests {
		result := min(test.x, test.y)
		if result != test.result {
			t.Fatalf("get min fail, input: %v, %v, expected %v, got %v", test.x, test.y, test.result, result)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		x, y, result uint64
	}{
		{1, 2, 2},
		{1, 1, 1},
		{2, 1, 2},
	}
	for _, test := range tests {
		result := max(test.x, test.y)
		if result != test.result {
			t.Fatalf("get max fail, input: %v, %v, expected %v, got %v", test.x, test.y, test.result, result)
		}
	}
}

func TestSafeAdd(t *testing.T) {
	tests := []struct {
		x, y, result uint64
		overflow     bool
	}{
		{1, 2, 3, false},
		{maxUint64 - 1, 1, maxUint64, false},
		{maxUint64, 1, 0, true},
		{maxUint64, 2, 1, true},
	}
	for _, test := range tests {
		result, overflow := SafeAdd(test.x, test.y)
		if result != test.result || overflow != test.overflow {
			t.Fatalf("safe add fail, input: %v, %v, expected [%v, %v], got [%v, %v]", test.x, test.y, test.result, test.overflow, result, overflow)
		}
	}
}

func TestSafeMul(t *testing.T) {
	tests := []struct {
		x, y, result uint64
		overflow     bool
	}{
		{1, 0, 0, false},
		{0, 1, 0, false},
		{1, 2, 2, false},
		{maxUint64, 2, maxUint64 - 1, true},
		{maxUint64 / 2, 2, maxUint64 - 1, false},
	}
	for _, test := range tests {
		result, overflow := SafeMul(test.x, test.y)
		if result != test.result || overflow != test.overflow {
			t.Fatalf("safe add fail, input: %v, %v, expected [%v, %v], got [%v, %v]", test.x, test.y, test.result, test.overflow, result, overflow)
		}
	}
}

func TestBigPow(t *testing.T) {
	tests := []struct {
		x, y   int64
		result *big.Int
	}{
		{0, 0, big.NewInt(1)},
		{0, 2, big.NewInt(0)},
		{2, 0, big.NewInt(1)},
		{2, 4, big.NewInt(16)},
		{2, 33, big.NewInt(8589934592)},
		{2, 66, new(big.Int).Mul(big.NewInt(8589934592), big.NewInt(8589934592))},
	}
	for _, test := range tests {
		result := BigPow(test.x, test.y)
		if result.Cmp(test.result) != 0 {
			t.Fatalf("big pow fail, %v ** %v = expected %v, got %v", test.x, test.y, test.result, result)
		}
	}
}

func TestReadBits(t *testing.T) {
	tests := []struct {
		data        *big.Int
		buf, result []byte
	}{
		{tt256m1, make([]byte, 32), []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
		{tt256m1, make([]byte, 33), []byte{0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
		{tt256m1, make([]byte, 2), []byte{255, 255}},
		{Big0, make([]byte, 2), []byte{0, 0}},
		{Big0, make([]byte, 32), []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
	}
	for _, test := range tests {
		ReadBits(test.data, test.buf)
		if bytes.Compare(test.buf, test.result) != 0 {
			t.Fatalf("read bits fail, data: [%v], expected [%v], got [%v]", test.data, test.result, test.buf)
		}
	}
}

func TestU256(t *testing.T) {
	tests := []struct {
		input, result *big.Int
	}{
		{big.NewInt(0), big.NewInt(0)},
		{big.NewInt(1), big.NewInt(1)},
		{new(big.Int).Set(tt256m1), new(big.Int).Set(tt256m1)},
		{new(big.Int).Set(tt256), big.NewInt(0)},
		{new(big.Int).Add(tt256, Big1), big.NewInt(1)},
	}
	for _, test := range tests {
		result := U256(test.input)
		if result.Cmp(test.result) != 0 {
			t.Fatalf("get u256 fail, input: [%v], expected [%v], got [%v]", test.input, test.result, result)
		}
	}
}
