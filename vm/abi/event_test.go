package abi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
)

var jsonEventTransfer = []byte(`{
  "anonymous": false,
  "inputs": [
    {
      "indexed": true, "name": "from", "type": "address"
    }, {
      "indexed": true, "name": "to", "type": "address"
    }, {
      "indexed": false, "name": "value", "type": "uint256"
  }],
  "name": "Transfer",
  "type": "event"
}`)

var jsonEventStake = []byte(`{
  "anonymous": false,
  "inputs": [{
      "indexed": false, "name": "who", "type": "address"
    }, {
      "indexed": false, "name": "wad", "type": "uint128"
    }, {
      "indexed": false, "name": "currency", "type": "bytes3"
  }],
  "name": "Stake",
  "type": "event"
}`)

var jsonEventMixedCase = []byte(`{
	"anonymous": false,
	"inputs": [{
		"indexed": false, "name": "value", "type": "uint256"
	  }, {
		"indexed": false, "name": "_value", "type": "uint256"
	  }, {
		"indexed": false, "name": "Value", "type": "uint256"
	}],
	"name": "MixedCase",
	"type": "event"
  }`)

// 1000000
var eventTransferData1 = "00000000000000000000000000000000000000000000000000000000000f4240"

// "0x00Ce0d46d924CC8437c806721496599FC3FFA268", 2218516807680, "usd"
var eventStakeData1 = "000000000000000000000000ce0d46d924cc8437c806721496599fc3ffa268000000000000000000000000000000000000000000000000000000020489e800007573640000000000000000000000000000000000000000000000000000000000"

// 1000000,2218516807680,1000001
var eventMixedCaseData1 = "00000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000020489e8000000000000000000000000000000000000000000000000000000000000000f4241"

func TestEventId(t *testing.T) {
	var table = []struct {
		definition   string
		expectations map[string]types.Hash
	}{
		{
			definition: `[
			{ "type" : "event", "name" : "balance", "inputs": [{ "name" : "in", "type": "uint256" }] },
			{ "type" : "event", "name" : "check", "inputs": [{ "name" : "t", "type": "address" }, { "name": "b", "type": "uint256" }] }
			]`,
			expectations: map[string]types.Hash{
				"balance": types.DataHash([]byte("balance(uint256)")),
				"check":   types.DataHash([]byte("check(address,uint256)")),
			},
		},
	}

	for _, test := range table {
		abi, err := JSONToABIContract(strings.NewReader(test.definition))
		if err != nil {
			t.Fatal(err)
		}

		for name, event := range abi.Events {
			if event.Id() != test.expectations[name] {
				t.Errorf("expected id to be %x, got %x", test.expectations[name], event.Id())
			}
		}
	}
}

// TestEventMultiValueWithArrayUnpack verifies that array fields will be counted after parsing array.
func TestEventMultiValueWithArrayUnpack(t *testing.T) {
	definition := `[{"name": "test", "type": "event", "inputs": [{"indexed": false, "name":"value1", "type":"uint8[2]"},{"indexed": false, "name":"value2", "type":"uint8"}]}]`
	type testStruct struct {
		Value1 [2]uint8
		Value2 uint8
	}
	abi, err := JSONToABIContract(strings.NewReader(definition))
	require.NoError(t, err)
	var b bytes.Buffer
	var i uint8 = 1
	for ; i <= 3; i++ {
		packedi, err := packNum(reflect.ValueOf(i))
		if err != nil {
			t.Fatalf("pack number failed, %v", err)
		}
		b.Write(packedi)
	}
	var rst testStruct
	require.NoError(t, abi.UnpackEvent(&rst, "test", b.Bytes()))
	require.Equal(t, [2]uint8{1, 2}, rst.Value1)
	require.Equal(t, uint8(3), rst.Value2)
}

func TestEventTupleUnpack(t *testing.T) {

	type EventTransfer struct {
		Value *big.Int
	}

	type EventTransferWithTag struct {
		// this is valid because `value` is not exportable,
		// so value is only unmarshalled into `Value1`.
		value  *big.Int
		Value1 *big.Int `abi:"value"`
	}

	type BadEventTransferWithSameFieldAndTag struct {
		Value  *big.Int
		Value1 *big.Int `abi:"value"`
	}

	type BadEventTransferWithDuplicatedTag struct {
		Value1 *big.Int `abi:"value"`
		Value2 *big.Int `abi:"value"`
	}

	type BadEventTransferWithEmptyTag struct {
		Value *big.Int `abi:""`
	}

	type EventStake struct {
		Who      types.Address
		Wad      *big.Int
		Currency [3]byte
	}

	type BadEventStake struct {
		Who      string
		Wad      int
		Currency [3]byte
	}

	type EventMixedCase struct {
		Value1 *big.Int `abi:"value"`
		Value2 *big.Int `abi:"_value"`
		Value3 *big.Int `abi:"Value"`
	}

	bigint := new(big.Int)
	bigintExpected := big.NewInt(1000000)
	bigintExpected2 := big.NewInt(2218516807680)
	bigintExpected3 := big.NewInt(1000001)
	addr, _ := types.BytesToAddress(helper.HexToBytes("00Ce0d46d924CC8437c806721496599FC3FFA26800"))
	var testCases = []struct {
		data     string
		dest     interface{}
		expected interface{}
		jsonLog  []byte
		error    string
		name     string
	}{{
		eventTransferData1,
		&EventTransfer{},
		&EventTransfer{Value: bigintExpected},
		jsonEventTransfer,
		"",
		"Can unpack ERC20 Transfer event into structure",
	}, {
		eventTransferData1,
		&EventTransferWithTag{},
		&EventTransferWithTag{Value1: bigintExpected},
		jsonEventTransfer,
		"",
		"Can unpack ERC20 Transfer event into structure with abi: tag",
	}, {
		eventTransferData1,
		&BadEventTransferWithDuplicatedTag{},
		&BadEventTransferWithDuplicatedTag{},
		jsonEventTransfer,
		"struct: abi tag in 'Value2' already mapped",
		"Can not unpack ERC20 Transfer event with duplicated abi tag",
	}, {
		eventTransferData1,
		&BadEventTransferWithSameFieldAndTag{},
		&BadEventTransferWithSameFieldAndTag{},
		jsonEventTransfer,
		"abi: multiple variables maps to the same abi field 'value'",
		"Can not unpack ERC20 Transfer event with a field and a tag mapping to the same abi variable",
	}, {
		eventTransferData1,
		&BadEventTransferWithEmptyTag{},
		&BadEventTransferWithEmptyTag{},
		jsonEventTransfer,
		"struct: abi tag in 'Value' is empty",
		"Can not unpack ERC20 Transfer event with an empty tag",
	}, {
		eventStakeData1,
		&EventStake{},
		&EventStake{
			addr,
			bigintExpected2,
			[3]byte{'u', 's', 'd'}},
		jsonEventStake,
		"",
		"Can unpack Stake event into structure",
	}, {
		eventStakeData1,
		&[]interface{}{&types.Address{}, &bigint, &[3]byte{}},
		&[]interface{}{
			&addr,
			&bigintExpected2,
			&[3]byte{'u', 's', 'd'}},
		jsonEventStake,
		"",
		"Can unpack Stake event into slice",
	}, {
		eventStakeData1,
		&[3]interface{}{&types.Address{}, &bigint, &[3]byte{}},
		&[3]interface{}{
			&addr,
			&bigintExpected2,
			&[3]byte{'u', 's', 'd'}},
		jsonEventStake,
		"",
		"Can unpack Stake event into an array",
	}, {
		eventStakeData1,
		&[]interface{}{new(int), 0, 0},
		&[]interface{}{},
		jsonEventStake,
		"abi: cannot unmarshal types.Address in to int",
		"Can not unpack Stake event into slice with wrong types",
	}, {
		eventStakeData1,
		&BadEventStake{},
		&BadEventStake{},
		jsonEventStake,
		"abi: cannot unmarshal types.Address in to string",
		"Can not unpack Stake event into struct with wrong filed types",
	}, {
		eventStakeData1,
		&[]interface{}{types.Address{}, new(big.Int)},
		&[]interface{}{},
		jsonEventStake,
		"abi: insufficient number of elements in the list/array for unpack, want 3, got 2",
		"Can not unpack Stake event into too short slice",
	}, {
		eventStakeData1,
		new(map[string]interface{}),
		&[]interface{}{},
		jsonEventStake,
		"abi: cannot unmarshal tuple into map[string]interface {}",
		"Can not unpack Stake event into map",
	}, {
		eventMixedCaseData1,
		&EventMixedCase{},
		&EventMixedCase{Value1: bigintExpected, Value2: bigintExpected2, Value3: bigintExpected3},
		jsonEventMixedCase,
		"",
		"Can unpack abi variables with mixed case",
	}}

	for _, tc := range testCases {
		assert := assert.New(t)
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := unpackTestEventData(tc.dest, tc.data, tc.jsonLog, assert)
			if tc.error == "" {
				assert.Nil(err, tc.name)
				assert.Equal(tc.expected, tc.dest, tc.name)
			} else {
				assert.EqualError(err, tc.error, tc.name)
			}
		})
	}
}

func unpackTestEventData(dest interface{}, hexData string, jsonEvent []byte, assert *assert.Assertions) error {
	data, err := hex.DecodeString(hexData)
	assert.NoError(err, "Hex data should be a correct hex-string")
	var e Event
	assert.NoError(json.Unmarshal(jsonEvent, &e), "Should be able to unmarshal event ABI")
	a := ABIContract{Events: map[string]Event{"e": e}}
	return a.UnpackEvent(dest, "e", data)
}

// TestEventUnpackIndexed verifies that indexed field will be skipped by event decoder.
func TestEventUnpackIndexed(t *testing.T) {
	definition := `[{"name": "test", "type": "event", "inputs": [{"indexed": true, "name":"value1", "type":"uint8"},{"indexed": false, "name":"value2", "type":"uint8"}]}]`
	type testStruct struct {
		Value1 uint8
		Value2 uint8
	}
	abi, err := JSONToABIContract(strings.NewReader(definition))
	require.NoError(t, err)
	var b bytes.Buffer
	packedi, err := packNum(reflect.ValueOf(uint8(8)))
	if err != nil {
		t.Fatalf("pack number failed, %v", err)
	}
	b.Write(packedi)
	var rst testStruct
	require.NoError(t, abi.UnpackEvent(&rst, "test", b.Bytes()))
	require.Equal(t, uint8(0), rst.Value1)
	require.Equal(t, uint8(8), rst.Value2)
}

// TestEventIndexedWithArrayUnpack verifies that decoder will not overlow when static array is indexed input.
func TestEventIndexedWithArrayUnpack(t *testing.T) {
	definition := `[{"name": "test", "type": "event", "inputs": [{"indexed": true, "name":"value1", "type":"uint8[2]"},{"indexed": false, "name":"value2", "type":"string"}]}]`
	type testStruct struct {
		Value1 [2]uint8
		Value2 string
	}
	abi, err := JSONToABIContract(strings.NewReader(definition))
	require.NoError(t, err)
	var b bytes.Buffer
	stringOut := "abc"
	// number of fields that will be encoded * 32
	packed32, err := packNum(reflect.ValueOf(32))
	if err != nil {
		t.Fatalf("pack number failed, %v", err)
	}
	packedout, err := packNum(reflect.ValueOf(len(stringOut)))
	if err != nil {
		t.Fatalf("pack number failed, %v", err)
	}
	b.Write(packed32)
	b.Write(packedout)
	b.Write(helper.RightPadBytes([]byte(stringOut), helper.WordSize))
	var rst testStruct
	require.NoError(t, abi.UnpackEvent(&rst, "test", b.Bytes()))
	require.Equal(t, [2]uint8{0, 0}, rst.Value1)
	require.Equal(t, stringOut, rst.Value2)
}

func TestDirectUnpackEvent(t *testing.T) {
	definition := `[{"name": "test", "type": "event", "inputs": [{"indexed": true, "name":"value1", "type":"uint8"},{"indexed": true, "name":"value2", "type":"string"},{"indexed": false, "name":"value3", "type":"uint8"}]}]`
	abi, err := JSONToABIContract(strings.NewReader(definition))
	require.NoError(t, err)
	value1 := uint8(1)
	value2 := "abc"
	value3 := uint8(2)
	topics, data, err := abi.PackEvent("test", value1, value2, value3)
	if err != nil {
		t.Fatalf("pack number failed, %v", err)
	}

	name, result, err := abi.DirectUnpackEvent(topics, data)
	if err != nil {
		t.Fatalf("pack number failed, %v", err)
	}
	hexValue2, _ := types.HexToHash("b728bc01f43abb5cfc587f771a2a7946995125cc827fa4d4232536457867b1b0")
	if name != "event test(uint8 indexed value1, string indexed value2, uint8 value3)" || len(result) != 3 || result[0] != value1 || result[1] != hexValue2 || result[2] != value3 {
		t.Fatalf("unpack event failed, got %v", result)
	}
}
