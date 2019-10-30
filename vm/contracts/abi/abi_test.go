package abi

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
	"testing"
)

func TestContractsABIInit(t *testing.T) {
	tests := []string{jsonQuota, jsonGovernance, jsonAsset}
	for _, data := range tests {
		if _, err := abi.JSONToABIContract(strings.NewReader(data)); err != nil {
			t.Fatalf("json to abi failed, %v, %v", data, err)
		}
	}
}

var (
	t1 = types.TokenTypeId{0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	t2 = types.TokenTypeId{0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	t3 = types.TokenTypeId{0, 0, 0, 0, 0, 0, 0, 0, 0, 3}
)

func TestDeleteTokenId(t *testing.T) {
	tests := []struct {
		input   []types.TokenTypeId
		tokenID types.TokenTypeId
		output  []types.TokenTypeId
	}{
		{[]types.TokenTypeId{t1}, t1, []types.TokenTypeId{}},
		{[]types.TokenTypeId{t1}, t2, []types.TokenTypeId{t1}},
		{[]types.TokenTypeId{t1, t2}, t1, []types.TokenTypeId{t2}},
		{[]types.TokenTypeId{t1, t2}, t2, []types.TokenTypeId{t1}},
		{[]types.TokenTypeId{t1, t2}, t3, []types.TokenTypeId{t1, t2}},
		{[]types.TokenTypeId{t1, t2, t3}, t1, []types.TokenTypeId{t2, t3}},
		{[]types.TokenTypeId{t1, t2, t3}, t2, []types.TokenTypeId{t1, t3}},
		{[]types.TokenTypeId{t1, t2, t3}, t3, []types.TokenTypeId{t1, t2}},
	}
	for _, test := range tests {
		var idList []byte
		for _, tid := range test.input {
			idList = AppendTokenID(idList, tid)
		}
		result := DeleteTokenID(idList, test.tokenID)
		var target []byte
		for _, tid := range test.output {
			target = AppendTokenID(target, tid)
		}
		if !bytes.Equal(result, target) {
			t.Fatalf("delete token id failed, delete %v from input %v, expected %v, got %v", test.tokenID, test.input, target, result)
		}
	}
}

func TestABIContract_EventById(t *testing.T) {
	for _, e := range ABIAsset.Events {
		fmt.Printf("%v: %v\n", e.Name, e.Id().String())
	}
}

func TestABIContract_MethodById(t *testing.T) {
	for _, m := range ABIGovernance.Methods {
		fmt.Printf("%v: %v\n", m.Sig(), hex.EncodeToString(m.Id()))
	}
}

func TestABIContract_CallbackById(t *testing.T) {
	for _, m := range ABIQuota.Callbacks {
		fmt.Printf("%v: %v\n", m.Sig(), hex.EncodeToString(m.Id()))
	}
}

func TestRegistrationInfoVariable(t *testing.T) {
	v, err := ABIGovernance.PackVariable(VariableNameRegistrationInfo, "s1", types.Address{}, types.Address{}, big.NewInt(10), uint64(100), int64(1000), int64(2000), []types.Address{})
	if err != nil {
		t.Fatalf("pack registration info failed %v", err)
	}
	if !bytes.Equal(v[:32], registerInfoValuePrefix) {
		t.Fatalf("check registration info prefix failed, expected %v, got %v", hex.EncodeToString(registerInfoValuePrefix), hex.EncodeToString(v[:32]))
	}
	v2, err2 := ABIGovernance.PackVariable(VariableNameRegistrationInfoV2, "s1", types.Address{}, types.Address{}, types.Address{}, big.NewInt(10), uint64(100), int64(1000), int64(2000), []types.Address{})
	if err2 != nil {
		t.Fatalf("pack registration info v2 failed %v", err2)
	}
	if bytes.Equal(v2[:32], registerInfoValuePrefix) {
		t.Fatalf("check registration info v2 prefix failed, expected not equal to %v, got %v", hex.EncodeToString(registerInfoValuePrefix), hex.EncodeToString(v2[:32]))
	}
}
