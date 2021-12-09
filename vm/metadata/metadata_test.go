package metadata

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"reflect"
	"testing"
)

var (
	Hash = types.Hash{97, 183, 48, 235, 101, 233, 99, 47, 158, 102, 219, 185, 87, 54, 55, 226, 222, 80, 17, 178, 245, 18, 137, 74, 55, 152, 28, 98, 222, 60, 189, 64}
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

func TestHasMetaData(t *testing.T) {
	tests := []struct {
		in string
		out bool
	}{
		{ "00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108", true},
		{ "0000000076010200000000000000000000000000000000000000000000000000000000000000001122334401020304", true},
		{ "00000000760101", false},
		{ "000000007601", false},
		{ "aabbccdd", false},
		{ "aabbccdd00000000", false},
		{ "", false},
	}

	for _, test := range tests {
		data, _ :=  hex.DecodeString(test.in)

		assert.NotPanics(t, func() {
			HasMetaData(data)
		})

		got := HasMetaData(data)

		if got != test.out {
			t.Errorf("HasMetaData failed, input: %v, expected [%v], but got [%v]", test.in, test.out, got)
		}
	}
}

func TestGetSendType(t *testing.T) {
	tests := []struct {
		in string
		out SendType
	}{
		{ "00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108", SendType(SyncCall)},
		{ "0000000076010200000000000000000000000000000000000000000000000000000000000000001122334401020304", SendType(Callback)},
		{ "0000000076010200000000", SendType(Callback)},
		{ "00000000760101", SendType(AsyncCall)},
		{ "000000007601", SendType(AsyncCall)},
		{ "aabbccdd", SendType(AsyncCall)},
		{ "aabbccdd00000000", SendType(AsyncCall)},
		{ "", SendType(AsyncCall)},
	}

	for _, test := range tests {
		data, _ :=  hex.DecodeString(test.in)

		assert.NotPanics(t, func() {
			GetSendType(data)
		})

		got := GetSendType(data)

		if got != test.out {
			t.Errorf("TestGetSendType failed, input: %v, expected [%v], but got [%v]", test.in, test.out, got)
		}
	}
}

func TestGetMetaDataLength(t *testing.T) {
	tests := []struct {
		in string
		out int
	}{
		{ "00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108", 77},
		{ "0000000076010200000000000000000000000000000000000000000000000000000000000000001122334401020304", 39},
		{ "00000000760101", 0},
		{ "000000007601", 0},
		{ "aabbccdd", 0},
		{ "aabbccdd00000000", 0},
		{ "", 0},
	}

	for _, test := range tests {
		data, _ :=  hex.DecodeString(test.in)

		assert.NotPanics(t, func() {
			GetMetaDataLength(data)
		})

		got := GetMetaDataLength(data)

		if got != test.out {
			t.Errorf("GetMetaDataLength failed, input: %v, expected [%v], but got [%v]", test.in, test.out, got)
		}
	}
}

func TestGetReferencedSendHash(t *testing.T) {
	tests := []struct {
		in string
		out string
		err string
	}{
		{
			"00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108",
			Hash.String(),
			"",
		},
		{
			"0000000076010200000000000000000000000000000000000000000000000000000000000000001122334401020304",
			types.Hash{}.String(),
			"",
		},
		{
			"00000000760101",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"invalid metadata",
		},
		{
			"aabbccdd",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"invalid metadata",
		},
		{
			"",
			"0000000000000000000000000000000000000000000000000000000000000000",
			"invalid metadata",
		},
	}

	for _, test := range tests {
		data, _ :=  hex.DecodeString(test.in)

		assert.NotPanics(t, func() {
			GetReferencedSendHash(data)
		})

		got, err := GetReferencedSendHash(data)

		if err != nil && err.Error() != test.err {
			t.Errorf("GetReferencedSendHash failed, input:\n%v\nexpected error:\n%v\nbut got:\n%v", test.in, test.err, err)
		}

		if got.String() != test.out  {
			t.Errorf("GetReferencedSendHash failed, input:\n%v\nexpected:\n%v\nbut got:\n%v", test.in, test.out, got)
		}
	}
}

func TestGetCalldata(t *testing.T) {
	tests := []struct {
		in string
		out string
	}{
		{
			"00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108" +
				"656dfc6e0000000000000000000000000000000000000000000000000000000000000000",
			"656dfc6e0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			"0000000076010200000000000000000000000000000000000000000000000000000000000000001122334401020304",
			"1122334401020304",
		},
		{
			"00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108",
			"",
		},
		{
			"00000000760101",
			"00000000760101",
		},
		{
			"aabbccdd",
			"aabbccdd",
		},
		{
			"",
			"",
		},
	}

	for _, test := range tests {
		data, _ :=  hex.DecodeString(test.in)

		assert.NotPanics(t, func() {
			GetCalldata(data)
		})

		got := GetCalldata(data)


		expected, _ := hex.DecodeString(test.out)
		if !bytes.Equal(got, expected) {
			t.Errorf("GetCalldata failed, input:\n%v\nexpected:\n%v\nbut got:\n%v", test.in, hex.EncodeToString(expected), hex.EncodeToString(got))
		}
	}
}

func TestGetContext(t *testing.T) {
	tests := []struct {
		in string
		out *ExecutionContext
	}{
		{
			"00000000760101012061b730eb65e9632f9e66dbb9573637e2de5011b2f512894a37981c62de3cbd40112233440006010101020103011600000201800001010102010301040105010601070108",
			&ExecutionContext {
				[]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
				mockMemory(),
			},
		},
		{
			"0000000076010200000000000000000000000000000000000000000000000000000000000000001122334401020304",
			nil,
		},
		{
			"00000000760101000000",
			nil,
		},
		{
			"11223344",
			nil,
		},
		{
			"",
			nil,
		},
	}

	for _, test := range tests {
		data, _ :=  hex.DecodeString(test.in)

		assert.NotPanics(t, func() {
			GetContext(data)
		})

		got := GetContext(data)

		if !reflect.DeepEqual(got, test.out) {
			t.Errorf("TestGetContext failed. \ninput: \n%v\nexpected:\n%v\nbut got:\n%v", test.in, test.out, got)
		}
	}
}

func TestMarshalStack(t *testing.T) {
	tests := []struct {
		in []*big.Int
		out string
	}{
		{
			[]*big.Int{big.NewInt(1)},
			"0002" +
				"0101",
		},
		{
			[]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
			"0006" +
				"0101" +
				"0102" +
				"0103",
		},
		{
			[]*big.Int{},
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
	bigNumber, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	tests := [][]*big.Int {
		{big.NewInt(1)},
		{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
		{big.NewInt(1), big.NewInt(0).SetBytes(bigNumber), big.NewInt(3)},
		{},
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
		{},
	}

	for _, test := range tests {
		assert.NotPanics(t, func() {
			dump := MarshalMemory(test)
			UnmarshalMemory(dump)
		})

		dump := MarshalMemory(test)
		got := UnmarshalMemory(dump)

		if !bytes.Equal(got, test) {
			t.Errorf("TestUnmarshalMemory failed. \nmemory: \n%v\ndump:\n%v\nbut got:\n%v", test, dump, got)
		}
	}
}

func TestMarshalContext(t *testing.T) {
	bigNumber, _ := hex.DecodeString("55c92b33cfcb3726405844155ad76a6dccf25f2401")
	mem := mockMemory()
	tests := []*ExecutionContext{
		{
			Stack: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
			Memory: []byte{0, 0, 0, 0, 0x11, 0x22, 0x33, 0x44},
		},
		{
			Stack: []*big.Int{big.NewInt(1), big.NewInt(0).SetBytes(bigNumber), big.NewInt(3)},
			Memory: mem,
		},
		{
			Stack: []*big.Int{},
			Memory: mem,
		},
	}


	for _, test := range tests {
		assert.NotPanics(t, func() {
			MarshalContext(test)
		})

		dump := MarshalContext(test)
		fmt.Printf("stack: %v\nmemory: %v\ndump: %v\n", test.Stack, test.Memory, hex.EncodeToString(dump))
	}
}

func TestMarshalAndUnmarshalContext(t *testing.T) {
	bigNumber, _ := hex.DecodeString("55c92b33cfcb3726405844155ad76a6dccf25f2401")
	mem := mockMemory()

	tests := []*ExecutionContext{
		{
			Stack: []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
			Memory: mem,
		},
		{
			Stack: []*big.Int{big.NewInt(1), big.NewInt(0).SetBytes(bigNumber), big.NewInt(3)},
			Memory: mem,
		},
		{
			Stack: []*big.Int{},
			Memory: mem,
		},
	}

	for _, test := range tests {
		assert.NotPanics(t, func() {
			dump := MarshalContext(test)
			UnmarshalContext(dump)
		})

		dump := MarshalContext(test)
		got := UnmarshalContext(dump)
		if !reflect.DeepEqual(got, test) {
			t.Errorf("TestUnmarshalContext failed. \ncontext: \n%v\ndump:\n%v\nbut got:\n%v", test, dump, got)
		}
	}
}

func TestEncodeSyncCallData(t *testing.T) {
	callback := big.NewInt(0x11223344)
	stack := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	mem := mockMemory()
	context := &ExecutionContext{stack, mem}
	calldata := []byte{0xa, 0xb, 0xc ,0xd}

	assert.NotPanics(t, func() {
		EncodeSyncCallData(Hash, callback, context, calldata)
	})

	data, err := EncodeSyncCallData(Hash, callback, context, calldata)
	fmt.Printf("data encoded: %v\n", hex.EncodeToString(data))
	assert.Equal(t, nil, err)
	assert.Equal(t,  SendType(SyncCall), GetSendType(data))

	hashGot, _ := GetReferencedSendHash(data)
	assert.Equal(t, Hash, hashGot)

	contextGot := GetContext(data)
	assert.Equal(t, context, contextGot)
	assert.Equal(t, stack, contextGot.Stack)
	assert.Equal(t, mem, contextGot.Memory)
}

func TestEncodeCallbackData(t *testing.T) {
	callback := big.NewInt(0x11223344)
	stack := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	mem := mockMemory()
	context := &ExecutionContext{stack, mem}

	sendData, _ := EncodeSyncCallData(Hash, callback, context, nil)

	sendBlock := util.MakeRequestBlock(
		types.Address{},
		types.Address{},
		ledger.BlockTypeSendCall,
		big.NewInt(0),
		types.TokenTypeId{},
		sendData)

	returnData := []byte{1, 2, 3, 4}

	assert.NotPanics(t, func() {
		EncodeCallbackData(sendBlock, returnData)
	})

	data, err := EncodeCallbackData(sendBlock, returnData)

	assert.Equal(t, nil, err)

	fmt.Printf("data encoded: %v\n", hex.EncodeToString(data))

	assert.Equal(t,  SendType(Callback), GetSendType(data))

	hashGot, _ := GetReferencedSendHash(data)
	assert.Equal(t, sendBlock.Hash, hashGot)
}


func BenchmarkMarshalStack(b *testing.B) {
	stack := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6), big.NewInt(7), big.NewInt(8)}
	for n := 0; n < b.N; n++ {
		MarshalStack(stack)
	}
}

func TestStackDumpSpaceCost(t *testing.T) {
	bigNumber, _ := hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	bigItem := big.NewInt(0).SetBytes(bigNumber)

	tests := [][]*big.Int {
		{big.NewInt(1)},
		{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
		{big.NewInt(1), bigItem, big.NewInt(3)},
		{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6)},
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
