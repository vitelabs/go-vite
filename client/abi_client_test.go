package client

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/vitelabs/go-vite/vm/abi"
	"gotest.tools/assert"

	"github.com/vitelabs/go-vite/common/types"
)

func TestAbiCli_CallOffChain(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	abi := ``
	offchainCode := ``
	contract := types.HexToAddressPanic("")

	abiCli, err := GetAbiCli(rpc, abi, offchainCode, contract)
	if err != nil {
		t.Fatal(err)
	}

	result, err := abiCli.CallOffChain("")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range result {
		t.Log(k, v)
	}

}

func TestUnpack(t *testing.T) {
	abiCode := ``
	contract, err := abi.JSONToABIContract(strings.NewReader(abiCode))
	assert.NilError(t, err)

	data, err := base64.StdEncoding.DecodeString("")
	assert.NilError(t, err)

	id, err := contract.MethodById(data[0:4])
	assert.NilError(t, err)
	t.Log(id)
	var inputs types.Address

	err = contract.UnpackMethod(&inputs, id.Name, data)
	assert.NilError(t, err)

	t.Log(inputs)

}
