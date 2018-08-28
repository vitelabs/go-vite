package vm

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestMemory(t *testing.T) {
	mem := newMemory()
	size := uint64(64)
	mem.resize(size)
	if len := mem.len(); len != 64 {
		t.Fatalf("memory resize error, expected %v, got %v", size, len)
	}

	data1, _ := hex.DecodeString("00CDEF090807060504030201ffffffffffffffffffffffffffffffffffffffff")
	mem.set(0, 32, data1)
	cpy := mem.get(0, 32)
	if bytes.Compare(cpy, data1) != 0 {
		t.Fatalf("memory get or set error, expected %v, got %v", data1, cpy)
	}

	ptr := mem.getPtr(0, 32)
	if bytes.Compare(ptr, data1) != 0 {
		t.Fatalf("memory get ptr error, expected %v, got %v", data1, ptr)
	}

	data2, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffABCDEF090807060504030201")
	for i := range data2 {
		ptr[i] = data2[i]
	}
	cpy2 := mem.get(0, 32)
	if bytes.Compare(cpy2, data2) != 0 {
		t.Fatalf("memory change ptr error")
	}

	mem.set32(0, tt255)
	cpy3 := mem.get(0, 32)
	if bytes.Compare(cpy3, tt255.Bytes()) != 0 {
		t.Fatalf("memory set32 error, expected %v, got %v", tt255.Bytes(), cpy3)
	}

	t.Log(mem.print())
}

func TestCalcMemSize(t *testing.T) {
	tests := []struct {
		off, l, result *big.Int
	}{
		{big.NewInt(0), big.NewInt(0), big.NewInt(0)},
		{big.NewInt(10), big.NewInt(0), big.NewInt(0)},
		{big.NewInt(10), big.NewInt(5), big.NewInt(15)},
	}
	for _, test := range tests {
		result := calcMemSize(test.off, test.l)
		if result.Cmp(test.result) != 0 {
			t.Fatalf("calc mem size error, off: %v, l: %v, expected %v, got %v", test.off, test.l, test.result, result)
		}
	}
}

func TestMemoryTable(t *testing.T) {
	tests := []struct {
		st     *stack
		op     func(stack *stack) *big.Int
		result *big.Int
	}{
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(64)}}, memoryBlake2b, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(0), big.NewInt(64)}}, memoryCallDataCopy, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(0), big.NewInt(64)}}, memoryCodeCopy, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(0), big.NewInt(64), big.NewInt(0)}}, memoryExtCodeCopy, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(0), big.NewInt(64)}}, memoryReturnDataCopy, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(64)}}, memoryMLoad, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(64)}}, memoryMStore, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(64)}}, memoryMStore8, big.NewInt(65)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(64)}}, memoryLog, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(64), big.NewInt(64), big.NewInt(32), big.NewInt(64), big.NewInt(0)}}, memoryDelegateCall, big.NewInt(128)},
		{&stack{data: []*big.Int{big.NewInt(64), big.NewInt(64), big.NewInt(32), big.NewInt(128), big.NewInt(0)}}, memoryDelegateCall, big.NewInt(160)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(64)}}, memoryReturn, big.NewInt(96)},
		{&stack{data: []*big.Int{big.NewInt(32), big.NewInt(64)}}, memoryRevert, big.NewInt(96)},
	}
	for i, test := range tests {
		result := test.op(test.st)
		if result.Cmp(test.result) != 0 {
			t.Fatalf("%v th get memory cost failed, stack: [%v], expected %v, got %v", i, test.st.print(), test.result, result)
		}
	}
}
