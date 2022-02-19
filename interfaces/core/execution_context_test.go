package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/types"
	"math/big"
	"reflect"
	"testing"
)

func mockMemory() []byte {
	// memory layout in the real world
	scratch := make([]byte, 64)
	zeroSlots := make([]byte, 32)
	heap, _ := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000001" +
			"0000000000000000000000000000000000000000000000000000000000000002" +
			"0000000000000000000000000000000000000000000000000000000000000003" +
			"0000000000000000000000000000000000000000000000000000000000000004" +
			"0000000000000000000000000000000000000000000000000000000000000005" +
			"0000000000000000000000000000000000000000000000000000000000000006" +
			"0000000000000000000000000000000000000000000000000000000000000007" +
			"0000000000000000000000000000000000000000000000000000000000000008")
	freePointer := 0x80 + len(heap)

	mem := make([]byte, 0, 64 + 32 + 32 + len(heap))
	mem = append(mem, scratch...)
	mem = append(mem, big.NewInt(int64(freePointer)).FillBytes(make([]byte, 32))...)
	mem = append(mem, zeroSlots...)
	mem = append(mem, heap...)

	return mem
}

func TestMarshalStack(t *testing.T) {
	tests := []struct {
		in []big.Int
		out string
	}{
		{
			[]big.Int{*big.NewInt(1)},
			"0002" +
				"0101",
		},
		{
			[]big.Int{*big.NewInt(1), *big.NewInt(2), *big.NewInt(3)},
			"0006" +
				"0101" +
				"0102" +
				"0103",
		},
		{
			[]big.Int{},
			"0000",
		},
		{
			nil,
			"0000",
		},
	}
	for _, test := range tests {
		assert.NotPanics(t, func() {
			MarshalStack(test.in)
		})

		got := MarshalStack(test.in)
		expected, _ := hex.DecodeString(test.out)
		if !bytes.Equal(got, expected) {
			t.Errorf("TestMarshalStack failed, input: %v\nexpected:\n%v\nbut got:\n%v", test.in, test.out, hex.EncodeToString(got))
		}
	}
}

func TestMarshalAndUnmarshalStack(t *testing.T) {
	bigNumber, _ := hex.DecodeString("1122334455667788112233445566778811223344556677881122334455667788")

	tests := [][]big.Int {
		{*big.NewInt(1)},
		{*big.NewInt(1), *big.NewInt(2), *big.NewInt(3)},
		{*big.NewInt(0).SetBytes(bigNumber)},
		{*big.NewInt(1), *big.NewInt(0).SetBytes(bigNumber), *big.NewInt(3)},
		nil,
	}

	for _, test := range tests {
		assert.NotPanics(t, func() {
			dump := MarshalStack(test)
			UnmarshalStack(dump)
		})

		dump := MarshalStack(test)
		got := UnmarshalStack(dump)
		if !reflect.DeepEqual(got, test) {
			t.Errorf("TestUnmarshalStack failed. \nstack: \n%v\ndump:\n%v\nbut got:\n%v", test, dump, got)
		}
	}
}

func TestMarshalMemory(t *testing.T) {
	mem := mockMemory()
	tests := [][]byte{
		mem,
		{0, 0, 0, 0, 0x11, 0x22, 0x33, 0x44},
		{},
		nil,
	}

	for _, test := range tests {
		assert.NotPanics(t, func() {
			MarshalMemory(test)
		})
		dump := MarshalMemory(test)
		fmt.Printf("memory: %v\ndump: %v\n", test, hex.EncodeToString(dump))
	}
}

func TestMarshalAndUnmarshalMemory(t *testing.T) {
	mem := mockMemory()

	tests := [][]byte{
		mem,
		{0, 0, 0, 0, 0x11, 0x22, 0x33, 0x44},
		nil,
	}

	for _, test := range tests {
		assert.NotPanics(t, func() {
			dump := MarshalMemory(test)
			UnmarshalMemory(dump)
		})

		dump := MarshalMemory(test)
		got := UnmarshalMemory(dump)
		assert.Equal(t, got, test)
		if !bytes.Equal(got, test) {
			t.Errorf("TestUnmarshalMemory failed. \nmemory: \n%v\ndump:\n%v\nbut got:\n%v", test, dump, got)
		}
	}
}

func Test_Serialize(t *testing.T) {
	tests := []ExecutionContext {
		{
			ReferrerSendHash: types.Hash{97, 183, 48, 235, 101, 233, 99, 47, 158, 102, 219, 185, 87, 54, 55, 226, 222, 80, 17, 178, 245, 18, 137, 74, 55, 152, 28, 98, 222, 60, 189, 64},
			CallbackId: *big.NewInt(1234),
			Stack: []big.Int{*big.NewInt(1), *big.NewInt(2), *big.NewInt(3)},
			Memory: mockMemory(),
		},
		{},
		{
			ReferrerSendHash: types.Hash{},
			CallbackId: *big.NewInt(0),
			Stack: nil,
			Memory: nil,
		},
	}

	for _, test := range tests {
		data, err := test.Serialize()
		fmt.Printf("data encoded: %v\n", hex.EncodeToString(data))

		assert.NoError(t, err)

		var context = ExecutionContext{}
		err = context.Deserialize(data)
		assert.NoError(t, err)

		assert.Equal(t, context, test)
	}
}

func BenchmarkMarshalStack(b *testing.B) {
	stack := []big.Int{*big.NewInt(1), *big.NewInt(2), *big.NewInt(3), *big.NewInt(4), *big.NewInt(5), *big.NewInt(6), *big.NewInt(7), *big.NewInt(8)}
	for n := 0; n < b.N; n++ {
		MarshalStack(stack)
	}
}

func TestStackDumpSpaceCost(t *testing.T) {
	bigNumber, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	bigItem := *big.NewInt(0).SetBytes(bigNumber)

	tests := [][]big.Int {
		{*big.NewInt(1)},
		{*big.NewInt(1), *big.NewInt(2), *big.NewInt(3)},
		{*big.NewInt(1), bigItem, *big.NewInt(3)},
		{*big.NewInt(1), *big.NewInt(2), *big.NewInt(3), *big.NewInt(4), *big.NewInt(5), *big.NewInt(6)},
		{bigItem, bigItem, bigItem, bigItem, bigItem},
	}

	for _, test := range tests {
		dump := MarshalStack(test)
		c := float32(len(dump)) / float32(len(test)*32)
		fmt.Printf("compression ratio = %v for %v\n", c, test)
	}
}

func TestMemoryDumpSpaceCost(t *testing.T) {
	scratch := make([]byte, 64)
	zeroSlots := make([]byte, 32)
	heap, _ := hex.DecodeString(
		"aabb000000000000000000000000000000000000000000000000000000000000" +
			"ccdd000000000000000000000000000000000000000000000000000000000000" +
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" +
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" +
			"aabb000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000006" +
			"0000000000000000000000000000000000000000000000000000000000000007" +
			"0000000000000000000000000000000000000000000000000000000000000008")
	freePointer := 0x80 + len(heap)

	mem := make([]byte, 0, 64 + 32 + 32 + len(heap))
	mem = append(mem, scratch...)
	mem = append(mem, big.NewInt(int64(freePointer)).FillBytes(make([]byte, 32))...)
	mem = append(mem, zeroSlots...)
	mem = append(mem, heap...)

	tests := [][]byte {
		mockMemory(),
		mem,
	}

	for _, test := range tests {
		dump := MarshalMemory(test)
		c := float32(len(dump)) / float32(len(test))
		fmt.Printf("compression ratio = %v for %v\n", c, hex.EncodeToString(test))
	}
}