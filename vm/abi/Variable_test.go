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
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
)

// TestVariableMultiValueWithArrayUnpack verifies that array fields will be counted after parsing array.
func TestVariableMultiValueWithArray(t *testing.T) {
	definition := `[{"name": "test", "type": "variable", "inputs": [{ "name":"value1", "type":"uint8[2]"},{ "name":"value2", "type":"uint8"}]}]`
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
	exp := testStruct{
		[2]uint8{1, 2},
		uint8(3),
	}
	var rst testStruct
	err = abi.UnpackVariable(&rst, "test", b.Bytes())
	if err != nil {
		t.Fatalf("unexpected error unpack variable %v", err)
	}
	if rst.Value1 != exp.Value1 || rst.Value2 != exp.Value2 {
		t.Fatalf("can not unpack test, got %v, expected %v", rst, exp)
	}

	packedData, err := abi.PackVariable("test", rst.Value1, rst.Value2)
	if err != nil {
		t.Fatalf("unexpected error pack variable %v", err)
	}
	if !bytes.Equal(packedData, b.Bytes()) {
		t.Fatalf("can not pack test, got %v, expected %v", packedData, b.Bytes())
	}
}

var jsonVariableTransfer = []byte(`{
  "inputs": [
    {
      "name": "from", "type": "address"
    }, {
      "name": "to", "type": "address"
    }, {
       "name": "value", "type": "uint256"
  }],
  "name": "Transfer",
  "type": "variable"
}`)

var jsonVariablePledge = []byte(`{
  "inputs": [{
       "name": "who", "type": "address"
    }, {
       "name": "wad", "type": "uint128"
    }, {
       "name": "currency", "type": "bytes3"
  }],
  "name": "Pledge",
  "type": "variable"
}`)

var jsonVariableMixedCase = []byte(`{
	"inputs": [{
		 "name": "value", "type": "uint256"
	  }, {
		 "name": "_value", "type": "uint256"
	  }, {
		 "name": "Value", "type": "uint256"
	}],
	"name": "MixedCase",
	"type": "variable"
  }`)

// 00Ce0d46d924CC8437c806721496599FC3FFA268, CA35B7D915458EF540ADE6068DFE2F44E8FA733C, 1000000
var variableTransferData1 = "00000000000000000000000000Ce0d46d924CC8437c806721496599FC3FFA268000000000000000000000000CA35B7D915458EF540ADE6068DFE2F44E8FA733C00000000000000000000000000000000000000000000000000000000000f4240"

// "0x00Ce0d46d924CC8437c806721496599FC3FFA268", 2218516807680, "usd"
var variablePledgeData1 = "00000000000000000000000000ce0d46d924cc8437c806721496599fc3ffa2680000000000000000000000000000000000000000000000000000020489e800007573640000000000000000000000000000000000000000000000000000000000"

// 1000000,2218516807680,1000001
var variableMixedCaseData1 = "00000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000020489e8000000000000000000000000000000000000000000000000000000000000000f4241"

func TestVariableTuple(t *testing.T) {

	type VariableTransfer struct {
		Value *big.Int
		From  types.Address
		To    types.Address
	}

	type VariableTransferWithTag struct {
		// this is valid because `value` is not exportable,
		// so value is only unmarshalled into `Value1`.
		value  *big.Int
		Value1 *big.Int `abi:"value"`
		From   types.Address
		To     types.Address
	}

	type BadVariableTransferWithSameFieldAndTag struct {
		Value  *big.Int
		Value1 *big.Int `abi:"value"`
		From   types.Address
		To     types.Address
	}

	type BadVariableTransferWithDuplicatedTag struct {
		Value1 *big.Int `abi:"value"`
		Value2 *big.Int `abi:"value"`
		From   types.Address
		To     types.Address
	}

	type BadVariableTransferWithEmptyTag struct {
		Value *big.Int `abi:""`
		From  types.Address
		To    types.Address
	}

	type VariablePledge struct {
		Who      types.Address
		Wad      *big.Int
		Currency [3]byte
	}

	type BadVariablePledge struct {
		Who      string
		Wad      int
		Currency [3]byte
	}

	type VariableMixedCase struct {
		Value1 *big.Int `abi:"value"`
		Value2 *big.Int `abi:"_value"`
		Value3 *big.Int `abi:"Value"`
	}

	typeBigint := new(big.Int)
	typeAddress1 := new(types.Address)
	typeAddress2 := new(types.Address)
	bigintExpected := big.NewInt(1000000)
	bigintExpected2 := big.NewInt(2218516807680)
	bigintExpected3 := big.NewInt(1000001)
	addr1, _ := types.BytesToAddress(util.HexToBytes("00Ce0d46d924CC8437c806721496599FC3FFA268"))
	addr2, _ := types.BytesToAddress(util.HexToBytes("CA35B7D915458EF540ADE6068DFE2F44E8FA733C"))
	var testCases = []struct {
		data     string
		dest     interface{}
		expected interface{}
		jsonLog  []byte
		error    string
		name     string
	}{{
		variableTransferData1,
		&VariableTransfer{},
		&VariableTransfer{Value: bigintExpected, From: addr1, To: addr2},
		jsonVariableTransfer,
		"",
		"Can unpack ERC20 Transfer variable into structure",
	}, {
		variableTransferData1,
		&[]interface{}{typeAddress1, typeAddress2, &typeBigint},
		&[]interface{}{&addr1, &addr2, &bigintExpected},
		jsonVariableTransfer,
		"",
		"Can unpack ERC20 Transfer variable into slice",
	}, {
		variableTransferData1,
		&VariableTransferWithTag{},
		&VariableTransferWithTag{Value1: bigintExpected, From: addr1, To: addr2},
		jsonVariableTransfer,
		"",
		"Can unpack ERC20 Transfer variable into structure with abi: tag",
	}, {
		variableTransferData1,
		&BadVariableTransferWithDuplicatedTag{},
		&BadVariableTransferWithDuplicatedTag{},
		jsonVariableTransfer,
		"struct: abi tag in 'Value2' already mapped",
		"Can not unpack ERC20 Transfer variable with duplicated abi tag",
	}, {
		variableTransferData1,
		&BadVariableTransferWithSameFieldAndTag{},
		&BadVariableTransferWithSameFieldAndTag{},
		jsonVariableTransfer,
		"abi: multiple variables maps to the same abi field 'value'",
		"Can not unpack ERC20 Transfer variable with a field and a tag mapping to the same abi variable",
	}, {
		variableTransferData1,
		&BadVariableTransferWithEmptyTag{},
		&BadVariableTransferWithEmptyTag{},
		jsonVariableTransfer,
		"struct: abi tag in 'Value' is empty",
		"Can not unpack ERC20 Transfer variable with an empty tag",
	}, {
		variablePledgeData1,
		&VariablePledge{},
		&VariablePledge{
			addr1,
			bigintExpected2,
			[3]byte{'u', 's', 'd'}},
		jsonVariablePledge,
		"",
		"Can unpack Pledge variable into structure",
	}, {
		variablePledgeData1,
		&[]interface{}{&types.Address{}, &typeBigint, &[3]byte{}},
		&[]interface{}{
			&addr1,
			&bigintExpected2,
			&[3]byte{'u', 's', 'd'}},
		jsonVariablePledge,
		"",
		"Can unpack Pledge variable into slice",
	}, {
		variablePledgeData1,
		&[3]interface{}{&types.Address{}, &typeBigint, &[3]byte{}},
		&[3]interface{}{
			&addr1,
			&bigintExpected2,
			&[3]byte{'u', 's', 'd'}},
		jsonVariablePledge,
		"",
		"Can unpack Pledge variable into an array",
	}, {
		variablePledgeData1,
		&[]interface{}{new(int), 0, 0},
		&[]interface{}{},
		jsonVariablePledge,
		"abi: cannot unmarshal types.Address in to int",
		"Can not unpack Pledge variable into slice with wrong types",
	}, {
		variablePledgeData1,
		&BadVariablePledge{},
		&BadVariablePledge{},
		jsonVariablePledge,
		"abi: cannot unmarshal types.Address in to string",
		"Can not unpack Pledge variable into struct with wrong filed types",
	}, {
		variablePledgeData1,
		&[]interface{}{types.Address{}, new(big.Int)},
		&[]interface{}{},
		jsonVariablePledge,
		"abi: insufficient number of elements in the list/array for unpack, want 3, got 2",
		"Can not unpack Pledge variable into too short slice",
	}, {
		variablePledgeData1,
		new(map[string]interface{}),
		&[]interface{}{},
		jsonVariablePledge,
		"abi: cannot unmarshal tuple into map[string]interface {}",
		"Can not unpack Pledge variable into map",
	}, {
		variableMixedCaseData1,
		&VariableMixedCase{},
		&VariableMixedCase{Value1: bigintExpected, Value2: bigintExpected2, Value3: bigintExpected3},
		jsonVariableMixedCase,
		"",
		"Can unpack abi variables with mixed case",
	}}

	for _, tc := range testCases {
		// TODO
		assert := assert.New(t)
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := unpackTestVariableData(tc.dest, tc.data, tc.jsonLog, assert)
			if tc.error == "" {
				if err != nil {
					t.Fatalf("Should be able to unpack variable data.")
				}
				assert.Equal(tc.expected, tc.dest, tc.name)
			} else {
				if err.Error() != tc.error {
					t.Fatalf(tc.name)
				}
			}
		})
	}
}

func unpackTestVariableData(dest interface{}, hexData string, jsonVariable []byte, assert *assert.Assertions) error {
	data, err := hex.DecodeString(hexData)
	assert.NoError(err, "Hex data should be a correct hex-string")
	var e Variable
	assert.NoError(json.Unmarshal(jsonVariable, &e), "Should be able to unmarshal variable ABI")
	a := ABIContract{Variables: map[string]Variable{"e": e}}
	return a.UnpackVariable(dest, "e", data)
}

type variableTestResult struct {
	Values [2]*big.Int
	Value1 *big.Int
	Value2 *big.Int
}

type variableTestCase struct {
	definition string
	want       variableTestResult
}

func (tc variableTestCase) encoded(intType, arrayType Type) []byte {
	var b bytes.Buffer
	if tc.want.Value1 != nil {
		val, _ := intType.pack(reflect.ValueOf(tc.want.Value1))
		b.Write(val)
	}

	if !reflect.DeepEqual(tc.want.Values, [2]*big.Int{nil, nil}) {
		val, _ := arrayType.pack(reflect.ValueOf(tc.want.Values))
		b.Write(val)
	}
	if tc.want.Value2 != nil {
		val, _ := intType.pack(reflect.ValueOf(tc.want.Value2))
		b.Write(val)
	}
	return b.Bytes()
}

// TestVariableUnpackIndexed verifies that indexed field will be skipped by variable decoder.
func TestVariableUnpackIndexed(t *testing.T) {
	definition := `[{"name": "test", "type": "variable", "inputs": [{"name":"value1", "type":"uint8"},{ "name":"value2", "type":"uint8"}]}]`
	type testStruct struct {
		Value1 uint8
		Value2 uint8
	}
	abi, err := JSONToABIContract(strings.NewReader(definition))
	require.NoError(t, err)
	var b bytes.Buffer
	b.Write(packNum(reflect.ValueOf(uint8(0))))
	b.Write(packNum(reflect.ValueOf(uint8(8))))
	var rst testStruct
	require.NoError(t, abi.UnpackVariable(&rst, "test", b.Bytes()))
	require.Equal(t, uint8(0), rst.Value1)
	require.Equal(t, uint8(8), rst.Value2)
}

// TestVariableIndexedWithArrayUnpack verifies that decoder will not overlow when static array is indexed input.
func TestVariableIndexedWithArrayUnpack(t *testing.T) {
	definition := `[{"name": "test", "type": "variable", "inputs": [{"name":"value1", "type":"uint8[2]"}, {"name":"value2","type":"string"}]}]`
	type testStruct struct {
		Value1 [2]uint8
		Value2 string
	}
	abi, err := JSONToABIContract(strings.NewReader(definition))
	require.NoError(t, err)
	stringOut := "abc"

	var b bytes.Buffer
	b.Write(packNum(reflect.ValueOf(1)))
	b.Write(packNum(reflect.ValueOf(2)))
	b.Write(packNum(reflect.ValueOf(96)))
	b.Write(packNum(reflect.ValueOf(len(stringOut))))
	b.Write(util.RightPadBytes([]byte(stringOut), 32))

	data, err := abi.PackVariable("test", [2]uint8{1, 2}, stringOut)
	if err != nil {
		t.Fatalf("unexpected error pack variable %v", err)
	}
	if !bytes.Equal(data, b.Bytes()) {
		t.Fatalf("pack variable failed, got %v, expected %v", data, b.Bytes())
	}
	var rst testStruct
	err = abi.UnpackVariable(&rst, "test", b.Bytes())
	if err != nil {
		t.Fatalf("unexpected error unpack variable %v", err)
	}
	if [2]uint8{1, 2} != rst.Value1 || stringOut != rst.Value2 {
		t.Fatalf("pack variable failed, got %v", rst)
	}
}
