package abi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
)

const jsondata = `
[
	{ "type" : "constructor", "inputs" : [ { "name" : "owner", "type" : "address" } ] },
	{ "type" : "function", "name" : "balance", "constant" : true },
	{ "type" : "function", "name" : "send", "constant" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] },
	{ "type" : "event", "name" : "Transfer", "anonymous" : false, "inputs" : [ { "indexed" : true, "name" : "from", "type" : "address" }, { "indexed" : true, "name" : "to", "type" : "address" }, { "name" : "value", "type" : "uint256" } ] },
	{ "type" : "variable", "name" : "register", "inputs" : [ { "name" : "node", "type" : "address" }, { "name" : "amount", "type" : "uint256" }, { "name" : "withdrawTime", "type" : "int64" }  ] }
]`

const jsondata2 = `
[
	{ "type" : "function", "name" : "balance", "constant" : true },
	{ "type" : "function", "name" : "send", "constant" : false, "inputs" : [ { "name" : "amount", "type" : "uint256" } ] },
	{ "type" : "function", "name" : "test", "constant" : false, "inputs" : [ { "name" : "number", "type" : "uint32" } ] },
	{ "type" : "function", "name" : "string", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "string" } ] },
	{ "type" : "function", "name" : "bool", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "bool" } ] },
	{ "type" : "function", "name" : "address", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "address" } ] },
	{ "type" : "function", "name" : "uint64[2]", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint64[2]" } ] },
	{ "type" : "function", "name" : "uint64[]", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint64[]" } ] },
	{ "type" : "function", "name" : "foo", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32" } ] },
	{ "type" : "function", "name" : "bar", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32" }, { "name" : "string", "type" : "uint16" } ] },
	{ "type" : "function", "name" : "slice", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint32[2]" } ] },
	{ "type" : "function", "name" : "slice256", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "uint256[2]" } ] },
	{ "type" : "function", "name" : "sliceAddress", "constant" : false, "inputs" : [ { "name" : "inputs", "type" : "address[]" } ] },
	{ "type" : "function", "name" : "sliceMultiAddress", "constant" : false, "inputs" : [ { "name" : "a", "type" : "address[]" }, { "name" : "b", "type" : "address[]" } ] }
]`

func TestReader(t *testing.T) {
	typeUint256, _ := NewType("uint256")
	typeInt64, _ := NewType("int64")
	typeAddress, _ := NewType("address")
	exp := ABIContract{
		Constructor: Method{
			"", false, []Argument{
				{"owner", typeAddress, false},
			},
		},
		Methods: map[string]Method{
			"balance": {
				"balance", true, nil,
			},
			"send": {
				"send", false, []Argument{
					{"amount", typeUint256, false},
				},
			},
		},
		Events: map[string]Event{
			"Transfer": {
				"Transfer", false, []Argument{
					{"from", typeAddress, true},
					{"to", typeAddress, true},
					{"value", typeUint256, false},
				},
			},
		},
		Variables: map[string]Variable{
			"register": {"register", []Argument{
				{"node", typeAddress, false},
				{"amount", typeUint256, false},
				{"withdrawTime", typeInt64, false},
			}},
		},
	}

	abi, err := JSONToABIContract(strings.NewReader(jsondata))
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(abi.Constructor, exp.Constructor) {
		t.Errorf("\nGot abi constructor: \n%v\ndoes not match expected constructor\n%v", abi.Constructor, exp.Constructor)
	}
	if len(exp.Methods) != len(abi.Methods) {
		t.Fatalf("\n Got abi method length %v does not match expected method length %v", len(abi.Methods), len(exp.Methods))
	}

	for name, expM := range exp.Methods {
		gotM, exist := abi.Methods[name]
		if !exist {
			t.Errorf("Missing expected method %v", name)
		}
		if !reflect.DeepEqual(gotM, expM) {
			t.Errorf("\nGot abi method: \n%v\ndoes not match expected method\n%v", gotM, expM)
		}
		gotM.String()
	}

	if len(exp.Events) != len(abi.Events) {
		t.Fatalf("\n Got abi event length %v does not match expected event length %v", len(abi.Events), len(exp.Events))
	}

	for name, expE := range exp.Events {
		gotE, exist := abi.Events[name]
		if !exist {
			t.Errorf("Missing expected event %v", name)
		}
		if !reflect.DeepEqual(gotE, expE) {
			t.Errorf("\nGot abi event: \n%v\ndoes not match expected event\n%v", gotE, expE)
		}
		gotE.String()
	}

	if len(exp.Variables) != len(abi.Variables) {
		t.Fatalf("\n Got abi variable length %v does not match expected variable length %v", len(abi.Variables), len(exp.Variables))
	}

	for name, expV := range exp.Variables {
		gotV, exist := abi.Variables[name]
		if !exist {
			t.Errorf("Missing expected variable %v", name)
		}
		if !reflect.DeepEqual(gotV, expV) {
			t.Errorf("\nGot abi evevariablent: \n%v\ndoes not match expected variable\n%v", gotV, expV)
		}
		gotV.String()
	}
}

func TestTestNumbers(t *testing.T) {
	abi, err := JSONToABIContract(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := abi.PackMethod("balance"); err != nil {
		t.Error(err)
	}

	if _, err := abi.PackMethod("balance", 1); err == nil {
		t.Error("expected error for balance(1)")
	}

	if _, err := abi.PackMethod("doesntexist", nil); err == nil {
		t.Errorf("doesntexist shouldn't exist")
	}

	if _, err := abi.PackMethod("doesntexist", 1); err == nil {
		t.Errorf("doesntexist(1) shouldn't exist")
	}

	if _, err := abi.PackMethod("send", big.NewInt(1000)); err != nil {
		t.Error(err)
	}

	i := new(int)
	*i = 1000
	if _, err := abi.PackMethod("send", i); err == nil {
		t.Errorf("expected send( ptr ) to throw, requires *big.Int instead of *int")
	}

	if _, err := abi.PackMethod("test", uint32(1000)); err != nil {
		t.Error(err)
	}
}

func TestTestString(t *testing.T) {
	abi, err := JSONToABIContract(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := abi.PackMethod("string", "hello world"); err != nil {
		t.Error(err)
	}
}

func TestTestBool(t *testing.T) {
	abi, err := JSONToABIContract(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if _, err := abi.PackMethod("bool", true); err != nil {
		t.Error(err)
	}
}

func TestTestSlice(t *testing.T) {
	abi, err := JSONToABIContract(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	slice := make([]uint64, 2)
	if _, err := abi.PackMethod("uint64[2]", slice); err != nil {
		t.Error(err)
	}

	if _, err := abi.PackMethod("uint64[]", slice); err != nil {
		t.Error(err)
	}
}

func TestMethodSignature(t *testing.T) {
	String, _ := NewType("string")
	m := Method{"foo", false, []Argument{{"bar", String, false}, {"baz", String, false}}}
	exp := "foo(string,string)"
	if m.Sig() != exp {
		t.Error("signature mismatch", exp, "!=", m.Sig())
	}

	idexp := types.DataHash([]byte(exp)).Bytes()[:4]
	if !bytes.Equal(m.Id(), idexp) {
		t.Errorf("expected ids to match %x != %x", m.Id(), idexp)
	}

	uintt, _ := NewType("uint256")
	m = Method{"foo", false, []Argument{{"bar", uintt, false}}}
	exp = "foo(uint256)"
	if m.Sig() != exp {
		t.Error("signature mismatch", exp, "!=", m.Sig())
	}
}

func TestMultiPack(t *testing.T) {
	abi, err := JSONToABIContract(strings.NewReader(jsondata2))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	sig := types.DataHash([]byte("bar(uint32,uint16)")).Bytes()[:4]
	sig = append(sig, make([]byte, 64)...)
	sig[35] = 10
	sig[67] = 11

	packed, err := abi.PackMethod("bar", uint32(10), uint16(11))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !bytes.Equal(packed, sig) {
		t.Errorf("expected %x got %x", sig, packed)
	}
}

func ExampleJSON() {
	const definition = `[{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"isBar","type":"function"}]`

	abi, err := JSONToABIContract(strings.NewReader(definition))
	if err != nil {
		log.Fatalln(err)
	}
	addr, _ := types.BytesToAddress(helper.LeftPadBytes(helper.HexToBytes("01"), types.AddressSize))
	out, err := abi.PackMethod("isBar", addr)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%x\n", out)
	// Output:
	// 65591c8e0000000000000000000000000000000000000000000000000000000000000001
}

func TestInputVariableInputLength(t *testing.T) {
	const definition = `[
	{ "type" : "function", "name" : "strOne", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" } ] },
	{ "type" : "function", "name" : "bytesOne", "constant" : true, "inputs" : [ { "name" : "str", "type" : "bytes" } ] },
	{ "type" : "function", "name" : "strTwo", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "str1", "type" : "string" } ] }
	]`

	abi, err := JSONToABIContract(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	// test one string
	strin := "hello world"
	strpack, err := abi.PackMethod("strOne", strin)
	if err != nil {
		t.Error(err)
	}

	offset := make([]byte, helper.WordSize)
	offset[helper.WordSize-1] = helper.WordSize
	length := make([]byte, helper.WordSize)
	length[helper.WordSize-1] = byte(len(strin))
	value := helper.RightPadBytes([]byte(strin), helper.WordSize)
	exp := append(offset, append(length, value...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	strpack = strpack[4:]
	if !bytes.Equal(strpack, exp) {
		t.Errorf("expected %x, got %x\n", exp, strpack)
	}

	// test one bytes
	btspack, err := abi.PackMethod("bytesOne", []byte(strin))
	if err != nil {
		t.Error(err)
	}
	// ignore first 4 bytes of the output. This is the function identifier
	btspack = btspack[4:]
	if !bytes.Equal(btspack, exp) {
		t.Errorf("expected %x, got %x\n", exp, btspack)
	}

	//  test two strings
	str1 := "hello"
	str2 := "world"
	str2pack, err := abi.PackMethod("strTwo", str1, str2)
	if err != nil {
		t.Error(err)
	}

	offset1 := make([]byte, helper.WordSize)
	offset1[helper.WordSize-1] = 64
	length1 := make([]byte, helper.WordSize)
	length1[helper.WordSize-1] = byte(len(str1))
	value1 := helper.RightPadBytes([]byte(str1), helper.WordSize)

	offset2 := make([]byte, helper.WordSize)
	offset2[helper.WordSize-1] = 128
	length2 := make([]byte, helper.WordSize)
	length2[helper.WordSize-1] = byte(len(str2))
	value2 := helper.RightPadBytes([]byte(str2), helper.WordSize)

	exp2 := append(offset1, offset2...)
	exp2 = append(exp2, append(length1, value1...)...)
	exp2 = append(exp2, append(length2, value2...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	str2pack = str2pack[4:]
	if !bytes.Equal(str2pack, exp2) {
		t.Errorf("expected %x, got %x\n", exp, str2pack)
	}

	// test two strings, first > 32, second < 32
	str1 = strings.Repeat("a", 33)
	str2pack, err = abi.PackMethod("strTwo", str1, str2)
	if err != nil {
		t.Error(err)
	}

	offset1 = make([]byte, helper.WordSize)
	offset1[helper.WordSize-1] = 64
	length1 = make([]byte, helper.WordSize)
	length1[helper.WordSize-1] = byte(len(str1))
	value1 = helper.RightPadBytes([]byte(str1), 64)
	offset2[helper.WordSize-1] = 160

	exp2 = append(offset1, offset2...)
	exp2 = append(exp2, append(length1, value1...)...)
	exp2 = append(exp2, append(length2, value2...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	str2pack = str2pack[4:]
	if !bytes.Equal(str2pack, exp2) {
		t.Errorf("expected %x, got %x\n", exp, str2pack)
	}

	// test two strings, first > 32, second >32
	str1 = strings.Repeat("a", 33)
	str2 = strings.Repeat("a", 33)
	str2pack, err = abi.PackMethod("strTwo", str1, str2)
	if err != nil {
		t.Error(err)
	}

	offset1 = make([]byte, helper.WordSize)
	offset1[helper.WordSize-1] = 64
	length1 = make([]byte, helper.WordSize)
	length1[helper.WordSize-1] = byte(len(str1))
	value1 = helper.RightPadBytes([]byte(str1), 64)

	offset2 = make([]byte, helper.WordSize)
	offset2[helper.WordSize-1] = 160
	length2 = make([]byte, helper.WordSize)
	length2[helper.WordSize-1] = byte(len(str2))
	value2 = helper.RightPadBytes([]byte(str2), 64)

	exp2 = append(offset1, offset2...)
	exp2 = append(exp2, append(length1, value1...)...)
	exp2 = append(exp2, append(length2, value2...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	str2pack = str2pack[4:]
	if !bytes.Equal(str2pack, exp2) {
		t.Errorf("expected %x, got %x\n", exp, str2pack)
	}
}

func TestInputFixedArrayAndVariableInputLength(t *testing.T) {
	const definition = `[
	{ "type" : "function", "name" : "fixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr", "type" : "uint256[2]" } ] },
	{ "type" : "function", "name" : "fixedArrBytes", "constant" : true, "inputs" : [ { "name" : "str", "type" : "bytes" }, { "name" : "fixedArr", "type" : "uint256[2]" } ] },
    { "type" : "function", "name" : "mixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr", "type": "uint256[2]" }, { "name" : "dynArr", "type": "uint256[]" } ] },
    { "type" : "function", "name" : "doubleFixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr1", "type": "uint256[2]" }, { "name" : "fixedArr2", "type": "uint256[3]" } ] },
    { "type" : "function", "name" : "multipleMixedArrStr", "constant" : true, "inputs" : [ { "name" : "str", "type" : "string" }, { "name" : "fixedArr1", "type": "uint256[2]" }, { "name" : "dynArr", "type" : "uint256[]" }, { "name" : "fixedArr2", "type" : "uint256[3]" } ] }
	]`

	abi, err := JSONToABIContract(strings.NewReader(definition))
	if err != nil {
		t.Error(err)
	}

	// test string, fixed array uint256[2]
	strin := "hello world"
	arrin := [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedArrStrPack, err := abi.PackMethod("fixedArrStr", strin, arrin)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	offset := make([]byte, helper.WordSize)
	offset[helper.WordSize-1] = 96
	length := make([]byte, helper.WordSize)
	length[helper.WordSize-1] = byte(len(strin))
	strvalue := helper.RightPadBytes([]byte(strin), helper.WordSize)
	arrinvalue1 := helper.LeftPadBytes(arrin[0].Bytes(), helper.WordSize)
	arrinvalue2 := helper.LeftPadBytes(arrin[1].Bytes(), helper.WordSize)
	exp := append(offset, arrinvalue1...)
	exp = append(exp, arrinvalue2...)
	exp = append(exp, append(length, strvalue...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	fixedArrStrPack = fixedArrStrPack[4:]
	if !bytes.Equal(fixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, fixedArrStrPack)
	}

	// test byte array, fixed array uint256[2]
	bytesin := []byte(strin)
	arrin = [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedArrBytesPack, err := abi.PackMethod("fixedArrBytes", bytesin, arrin)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	offset = make([]byte, helper.WordSize)
	offset[helper.WordSize-1] = 96
	length = make([]byte, helper.WordSize)
	length[helper.WordSize-1] = byte(len(strin))
	strvalue = helper.RightPadBytes([]byte(strin), helper.WordSize)
	arrinvalue1 = helper.LeftPadBytes(arrin[0].Bytes(), helper.WordSize)
	arrinvalue2 = helper.LeftPadBytes(arrin[1].Bytes(), helper.WordSize)
	exp = append(offset, arrinvalue1...)
	exp = append(exp, arrinvalue2...)
	exp = append(exp, append(length, strvalue...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	fixedArrBytesPack = fixedArrBytesPack[4:]
	if !bytes.Equal(fixedArrBytesPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, fixedArrBytesPack)
	}

	// test string, fixed array uint256[2], dynamic array uint256[]
	strin = "hello world"
	fixedarrin := [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	dynarrin := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	mixedArrStrPack, err := abi.PackMethod("mixedArrStr", strin, fixedarrin, dynarrin)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	stroffset := make([]byte, helper.WordSize)
	stroffset[helper.WordSize-1] = 128
	strlength := make([]byte, helper.WordSize)
	strlength[helper.WordSize-1] = byte(len(strin))
	strvalue = helper.RightPadBytes([]byte(strin), helper.WordSize)
	fixedarrinvalue1 := helper.LeftPadBytes(fixedarrin[0].Bytes(), helper.WordSize)
	fixedarrinvalue2 := helper.LeftPadBytes(fixedarrin[1].Bytes(), helper.WordSize)
	dynarroffset := make([]byte, helper.WordSize)
	dynarroffset[helper.WordSize-1] = byte(160 + ((len(strin)/helper.WordSize)+1)*helper.WordSize)
	dynarrlength := make([]byte, helper.WordSize)
	dynarrlength[helper.WordSize-1] = byte(len(dynarrin))
	dynarrinvalue1 := helper.LeftPadBytes(dynarrin[0].Bytes(), helper.WordSize)
	dynarrinvalue2 := helper.LeftPadBytes(dynarrin[1].Bytes(), helper.WordSize)
	dynarrinvalue3 := helper.LeftPadBytes(dynarrin[2].Bytes(), helper.WordSize)
	exp = append(stroffset, fixedarrinvalue1...)
	exp = append(exp, fixedarrinvalue2...)
	exp = append(exp, dynarroffset...)
	exp = append(exp, append(strlength, strvalue...)...)
	dynarrarg := append(dynarrlength, dynarrinvalue1...)
	dynarrarg = append(dynarrarg, dynarrinvalue2...)
	dynarrarg = append(dynarrarg, dynarrinvalue3...)
	exp = append(exp, dynarrarg...)

	// ignore first 4 bytes of the output. This is the function identifier
	mixedArrStrPack = mixedArrStrPack[4:]
	if !bytes.Equal(mixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, mixedArrStrPack)
	}

	// test string, fixed array uint256[2], fixed array uint256[3]
	strin = "hello world"
	fixedarrin1 := [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedarrin2 := [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	doubleFixedArrStrPack, err := abi.PackMethod("doubleFixedArrStr", strin, fixedarrin1, fixedarrin2)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	stroffset = make([]byte, helper.WordSize)
	stroffset[helper.WordSize-1] = 192
	strlength = make([]byte, helper.WordSize)
	strlength[helper.WordSize-1] = byte(len(strin))
	strvalue = helper.RightPadBytes([]byte(strin), helper.WordSize)
	fixedarrin1value1 := helper.LeftPadBytes(fixedarrin1[0].Bytes(), helper.WordSize)
	fixedarrin1value2 := helper.LeftPadBytes(fixedarrin1[1].Bytes(), helper.WordSize)
	fixedarrin2value1 := helper.LeftPadBytes(fixedarrin2[0].Bytes(), helper.WordSize)
	fixedarrin2value2 := helper.LeftPadBytes(fixedarrin2[1].Bytes(), helper.WordSize)
	fixedarrin2value3 := helper.LeftPadBytes(fixedarrin2[2].Bytes(), helper.WordSize)
	exp = append(stroffset, fixedarrin1value1...)
	exp = append(exp, fixedarrin1value2...)
	exp = append(exp, fixedarrin2value1...)
	exp = append(exp, fixedarrin2value2...)
	exp = append(exp, fixedarrin2value3...)
	exp = append(exp, append(strlength, strvalue...)...)

	// ignore first 4 bytes of the output. This is the function identifier
	doubleFixedArrStrPack = doubleFixedArrStrPack[4:]
	if !bytes.Equal(doubleFixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, doubleFixedArrStrPack)
	}

	// test string, fixed array uint256[2], dynamic array uint256[], fixed array uint256[3]
	strin = "hello world"
	fixedarrin1 = [2]*big.Int{big.NewInt(1), big.NewInt(2)}
	dynarrin = []*big.Int{big.NewInt(1), big.NewInt(2)}
	fixedarrin2 = [3]*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	multipleMixedArrStrPack, err := abi.PackMethod("multipleMixedArrStr", strin, fixedarrin1, dynarrin, fixedarrin2)
	if err != nil {
		t.Error(err)
	}

	// generate expected output
	stroffset = make([]byte, helper.WordSize)
	stroffset[helper.WordSize-1] = 224
	strlength = make([]byte, helper.WordSize)
	strlength[helper.WordSize-1] = byte(len(strin))
	strvalue = helper.RightPadBytes([]byte(strin), helper.WordSize)
	fixedarrin1value1 = helper.LeftPadBytes(fixedarrin1[0].Bytes(), helper.WordSize)
	fixedarrin1value2 = helper.LeftPadBytes(fixedarrin1[1].Bytes(), helper.WordSize)
	dynarroffset = U256(big.NewInt(int64(256 + ((len(strin)/helper.WordSize)+1)*helper.WordSize)))
	dynarrlength = make([]byte, helper.WordSize)
	dynarrlength[helper.WordSize-1] = byte(len(dynarrin))
	dynarrinvalue1 = helper.LeftPadBytes(dynarrin[0].Bytes(), helper.WordSize)
	dynarrinvalue2 = helper.LeftPadBytes(dynarrin[1].Bytes(), helper.WordSize)
	fixedarrin2value1 = helper.LeftPadBytes(fixedarrin2[0].Bytes(), helper.WordSize)
	fixedarrin2value2 = helper.LeftPadBytes(fixedarrin2[1].Bytes(), helper.WordSize)
	fixedarrin2value3 = helper.LeftPadBytes(fixedarrin2[2].Bytes(), helper.WordSize)
	exp = append(stroffset, fixedarrin1value1...)
	exp = append(exp, fixedarrin1value2...)
	exp = append(exp, dynarroffset...)
	exp = append(exp, fixedarrin2value1...)
	exp = append(exp, fixedarrin2value2...)
	exp = append(exp, fixedarrin2value3...)
	exp = append(exp, append(strlength, strvalue...)...)
	dynarrarg = append(dynarrlength, dynarrinvalue1...)
	dynarrarg = append(dynarrarg, dynarrinvalue2...)
	exp = append(exp, dynarrarg...)

	// ignore first 4 bytes of the output. This is the function identifier
	multipleMixedArrStrPack = multipleMixedArrStrPack[4:]
	if !bytes.Equal(multipleMixedArrStrPack, exp) {
		t.Errorf("expected %x, got %x\n", exp, multipleMixedArrStrPack)
	}
}

func TestDefaultFunctionParsing(t *testing.T) {
	const definition = `[{ "name" : "balance" }]`

	abi, err := JSONToABIContract(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := abi.Methods["balance"]; !ok {
		t.Error("expected 'balance' to be present")
	}
}

func TestBareEvents(t *testing.T) {
	const definition = `[
	{ "type" : "event", "name" : "balance" },
	{ "type" : "event", "name" : "anon", "anonymous" : true},
	{ "type" : "event", "name" : "args", "inputs" : [{ "indexed":false, "name":"arg0", "type":"uint256" }, { "indexed":true, "name":"arg1", "type":"address" }] }
	]`

	arg0, _ := NewType("uint256")
	arg1, _ := NewType("address")

	expectedEvents := map[string]struct {
		Anonymous bool
		Args      []Argument
	}{
		"balance": {false, nil},
		"anon":    {true, nil},
		"args": {false, []Argument{
			{Name: "arg0", Type: arg0, Indexed: false},
			{Name: "arg1", Type: arg1, Indexed: true},
		}},
	}

	abi, err := JSONToABIContract(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	if len(abi.Events) != len(expectedEvents) {
		t.Fatalf("invalid number of events after parsing, want %d, got %d", len(expectedEvents), len(abi.Events))
	}

	for name, exp := range expectedEvents {
		got, ok := abi.Events[name]
		if !ok {
			t.Errorf("could not found event %s", name)
			continue
		}
		if got.Anonymous != exp.Anonymous {
			t.Errorf("invalid anonymous indication for event %s, want %v, got %v", name, exp.Anonymous, got.Anonymous)
		}
		if len(got.Inputs) != len(exp.Args) {
			t.Errorf("invalid number of args, want %d, got %d", len(exp.Args), len(got.Inputs))
			continue
		}
		for i, arg := range exp.Args {
			if arg.Name != got.Inputs[i].Name {
				t.Errorf("events[%s].Input[%d] has an invalid name, want %s, got %s", name, i, arg.Name, got.Inputs[i].Name)
			}
			if arg.Indexed != got.Inputs[i].Indexed {
				t.Errorf("events[%s].Input[%d] has an invalid indexed indication, want %v, got %v", name, i, arg.Indexed, got.Inputs[i].Indexed)
			}
			if arg.Type.T != got.Inputs[i].Type.T {
				t.Errorf("events[%s].Input[%d] has an invalid type, want %x, got %x", name, i, arg.Type.T, got.Inputs[i].Type.T)
			}
		}
	}
}

func TestBareVariables(t *testing.T) {
	const definition1 = `[
	{ "type" : "variable", "name" : "balance" },
	{ "type" : "variable", "name" : "args", "inputs" : [{ "name":"arg0", "type":"uint256" }, { "name":"arg1", "type":"address" }] }
	]`
	_, err := JSONToABIContract(strings.NewReader(definition1))
	if err == nil {
		t.Fatalf("expected error")
	}

	const definition2 = `[
	{ "type" : "variable", "name" : "args", "inputs" : [{ "name":"arg0", "type":"uint256" }, { "name":"arg1", "type":"address" }] },
	{ "type" : "variable", "name" : "balance", "inputs" : [{ "name":"balance", "type":"uint256" }] }
	]`

	typeUint256, _ := NewType("uint256")
	typeAddress, _ := NewType("address")

	expectedVariables := map[string]struct {
		Args []Argument
	}{
		"args": {[]Argument{
			{Name: "arg0", Type: typeUint256, Indexed: false},
			{Name: "arg1", Type: typeAddress, Indexed: true},
		}},
		"balance": {[]Argument{
			{Name: "balance", Type: typeUint256, Indexed: false},
		}},
	}

	abi, err := JSONToABIContract(strings.NewReader(definition2))
	if err != nil {
		t.Fatal(err)
	}

	if len(abi.Variables) != len(expectedVariables) {
		t.Fatalf("invalid number of variables after parsing, want %d, got %d", len(expectedVariables), len(abi.Variables))
	}

	for name, exp := range expectedVariables {
		got, ok := abi.Variables[name]
		if !ok {
			t.Errorf("could not found variable %s", name)
			continue
		}
		if len(got.Inputs) != len(exp.Args) {
			t.Errorf("invalid number of args, want %d, got %d", len(exp.Args), len(got.Inputs))
			continue
		}
		for i, arg := range exp.Args {
			if arg.Name != got.Inputs[i].Name {
				t.Errorf("variables[%s].Input[%d] has an invalid name, want %s, got %s", name, i, arg.Name, got.Inputs[i].Name)
			}
			if arg.Type.T != got.Inputs[i].Type.T {
				t.Errorf("variables[%s].Input[%d] has an invalid type, want %x, got %x", name, i, arg.Type.T, got.Inputs[i].Type.T)
			}
		}
	}
}

// TestUnpackEvent is based on this contract:
//    contract T {
//      event received(address sender, uint amount, bytes memo);
//      event receivedAddr(address sender);
//      function receive(bytes memo) external payable {
//        received(msg.sender, msg.value, memo);
//        receivedAddr(msg.sender);
//      }
//    }
// When receive("X") is called with sender 0x00... and value 1, it produces this tx receipt:
//   receipt{status=1 cgas=23949 bloom=00000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000040200000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 logs=[log: b6818c8064f645cd82d99b59a1a267d6d61117ef [75fd880d39c1daf53b6547ab6cb59451fc6452d27caa90e5b6649dd8293b9eed] 000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158 9ae378b6d4409eada347a5dc0c180f186cb62dc68fcc0f043425eb917335aa28 0 95d429d309bb9d753954195fe2d69bd140b4ae731b9b5b605c34323de162cf00 0]}
func TestUnpackEvent(t *testing.T) {
	const abiJSON = `[{"constant":false,"inputs":[{"name":"memo","type":"bytes"}],"name":"receive","payable":true,"stateMutability":"payable","type":"function"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}],"name":"received","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"}],"name":"receivedAddr","type":"event"}]`
	abi, err := JSONToABIContract(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}

	const hexdata = `000000000000000000000000376c47978271565f56deb45495afa69e59c16ab200000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000158`
	data, err := hex.DecodeString(hexdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(data)%helper.WordSize == 0 {
		t.Errorf("len(data) is %d, want a non-multiple of 32", len(data))
	}

	type ReceivedEvent struct {
		Address types.Address
		Amount  *big.Int
		Memo    []byte
	}
	var ev ReceivedEvent

	err = abi.UnpackEvent(&ev, "received", data)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("len(data): %d; received event: %+v", len(data), ev)
	}

	type ReceivedAddrEvent struct {
		Address types.Address
	}
	var receivedAddrEv ReceivedAddrEvent
	err = abi.UnpackEvent(&receivedAddrEv, "receivedAddr", data)
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("len(data): %d; received event: %+v", len(data), receivedAddrEv)
	}
}

func TestABI_MethodById(t *testing.T) {
	const abiJSON = `[
		{"type":"function","name":"receive","constant":false,"inputs":[{"name":"memo","type":"bytes"}],"payable":true,"stateMutability":"payable"},
		{"type":"event","name":"received","anonymous":false,"inputs":[{"indexed":false,"name":"sender","type":"address"},{"indexed":false,"name":"amount","type":"uint256"},{"indexed":false,"name":"memo","type":"bytes"}]},
		{"type":"function","name":"fixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr","type":"uint256[2]"}]},
		{"type":"function","name":"fixedArrBytes","constant":true,"inputs":[{"name":"str","type":"bytes"},{"name":"fixedArr","type":"uint256[2]"}]},
		{"type":"function","name":"mixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr","type":"uint256[2]"},{"name":"dynArr","type":"uint256[]"}]},
		{"type":"function","name":"doubleFixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr1","type":"uint256[2]"},{"name":"fixedArr2","type":"uint256[3]"}]},
		{"type":"function","name":"multipleMixedArrStr","constant":true,"inputs":[{"name":"str","type":"string"},{"name":"fixedArr1","type":"uint256[2]"},{"name":"dynArr","type":"uint256[]"},{"name":"fixedArr2","type":"uint256[3]"}]},
		{"type":"function","name":"balance","constant":true},
		{"type":"function","name":"send","constant":false,"inputs":[{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"test","constant":false,"inputs":[{"name":"number","type":"uint32"}]},
		{"type":"function","name":"string","constant":false,"inputs":[{"name":"inputs","type":"string"}]},
		{"type":"function","name":"bool","constant":false,"inputs":[{"name":"inputs","type":"bool"}]},
		{"type":"function","name":"address","constant":false,"inputs":[{"name":"inputs","type":"address"}]},
		{"type":"function","name":"uint64[2]","constant":false,"inputs":[{"name":"inputs","type":"uint64[2]"}]},
		{"type":"function","name":"uint64[]","constant":false,"inputs":[{"name":"inputs","type":"uint64[]"}]},
		{"type":"function","name":"foo","constant":false,"inputs":[{"name":"inputs","type":"uint32"}]},
		{"type":"function","name":"bar","constant":false,"inputs":[{"name":"inputs","type":"uint32"},{"name":"string","type":"uint16"}]},
		{"type":"function","name":"_slice","constant":false,"inputs":[{"name":"inputs","type":"uint32[2]"}]},
		{"type":"function","name":"__slice256","constant":false,"inputs":[{"name":"inputs","type":"uint256[2]"}]},
		{"type":"function","name":"sliceAddress","constant":false,"inputs":[{"name":"inputs","type":"address[]"}]},
		{"type":"function","name":"sliceMultiAddress","constant":false,"inputs":[{"name":"a","type":"address[]"},{"name":"b","type":"address[]"}]}
	]
`
	abi, err := JSONToABIContract(strings.NewReader(abiJSON))
	if err != nil {
		t.Fatal(err)
	}
	for name, m := range abi.Methods {
		a := fmt.Sprintf("%v", m)
		m2, err := abi.MethodById(m.Id())
		if err != nil {
			t.Fatalf("Failed to look up ABI method: %v", err)
		}
		b := fmt.Sprintf("%v", m2)
		if a != b {
			t.Errorf("Method %v (id %v) not 'findable' by id in ABI", name, hex.EncodeToString(m.Id()))
		}
	}

}
