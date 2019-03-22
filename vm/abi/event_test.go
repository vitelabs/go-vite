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

var jsonEventPledge = []byte(`{
  "anonymous": false,
  "inputs": [{
      "indexed": false, "name": "who", "type": "address"
    }, {
      "indexed": false, "name": "wad", "type": "uint128"
    }, {
      "indexed": false, "name": "currency", "type": "bytes3"
  }],
  "name": "Pledge",
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
var eventPledgeData1 = "00000000000000000000000000ce0d46d924cc8437c806721496599fc3ffa2680000000000000000000000000000000000000000000000000000020489e800007573640000000000000000000000000000000000000000000000000000000000"

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
		b.Write(packNum(reflect.ValueOf(i)))
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

	type EventPledge struct {
		Who      types.Address
		Wad      *big.Int
		Currency [3]byte
	}

	type BadEventPledge struct {
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
	addr, _ := types.BytesToAddress(helper.HexToBytes("00Ce0d46d924CC8437c806721496599FC3FFA268"))
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
		&[]interface{}{&bigint},
		&[]interface{}{&bigintExpected},
		jsonEventTransfer,
		"",
		"Can unpack ERC20 Transfer event into slice",
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
		"Can not unpack ERC20 Transfer event with chain field and chain tag mapping to the same abi variable",
	}, {
		eventTransferData1,
		&BadEventTransferWithEmptyTag{},
		&BadEventTransferWithEmptyTag{},
		jsonEventTransfer,
		"struct: abi tag in 'Value' is empty",
		"Can not unpack ERC20 Transfer event with an empty tag",
	}, {
		eventPledgeData1,
		&EventPledge{},
		&EventPledge{
			addr,
			bigintExpected2,
			[3]byte{'u', 's', 'd'}},
		jsonEventPledge,
		"",
		"Can unpack Pledge event into structure",
	}, {
		eventPledgeData1,
		&[]interface{}{&types.Address{}, &bigint, &[3]byte{}},
		&[]interface{}{
			&addr,
			&bigintExpected2,
			&[3]byte{'u', 's', 'd'}},
		jsonEventPledge,
		"",
		"Can unpack Pledge event into slice",
	}, {
		eventPledgeData1,
		&[3]interface{}{&types.Address{}, &bigint, &[3]byte{}},
		&[3]interface{}{
			&addr,
			&bigintExpected2,
			&[3]byte{'u', 's', 'd'}},
		jsonEventPledge,
		"",
		"Can unpack Pledge event into an array",
	}, {
		eventPledgeData1,
		&[]interface{}{new(int), 0, 0},
		&[]interface{}{},
		jsonEventPledge,
		"abi: cannot unmarshal types.Address in to int",
		"Can not unpack Pledge event into slice with wrong types",
	}, {
		eventPledgeData1,
		&BadEventPledge{},
		&BadEventPledge{},
		jsonEventPledge,
		"abi: cannot unmarshal types.Address in to string",
		"Can not unpack Pledge event into struct with wrong filed types",
	}, {
		eventPledgeData1,
		&[]interface{}{types.Address{}, new(big.Int)},
		&[]interface{}{},
		jsonEventPledge,
		"abi: insufficient number of elements in the list/array for unpack, want 3, got 2",
		"Can not unpack Pledge event into too short slice",
	}, {
		eventPledgeData1,
		new(map[string]interface{}),
		&[]interface{}{},
		jsonEventPledge,
		"abi: cannot unmarshal tuple into map[string]interface {}",
		"Can not unpack Pledge event into map",
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
				assert.Nil(err, "Should be able to unpack event data.")
				assert.Equal(tc.expected, tc.dest, tc.name)
			} else {
				assert.EqualError(err, tc.error, tc.name)
			}
		})
	}
}

func unpackTestEventData(dest interface{}, hexData string, jsonEvent []byte, assert *assert.Assertions) error {
	data, err := hex.DecodeString(hexData)
	assert.NoError(err, "Hex data should be chain correct hex-string")
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
	b.Write(packNum(reflect.ValueOf(uint8(8))))
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
	b.Write(packNum(reflect.ValueOf(32)))
	b.Write(packNum(reflect.ValueOf(len(stringOut))))
	b.Write(helper.RightPadBytes([]byte(stringOut), helper.WordSize))
	var rst testStruct
	require.NoError(t, abi.UnpackEvent(&rst, "test", b.Bytes()))
	require.Equal(t, [2]uint8{0, 0}, rst.Value1)
	require.Equal(t, stringOut, rst.Value2)
}
