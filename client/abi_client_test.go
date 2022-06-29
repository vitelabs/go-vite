package client

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/vm/abi"
)

func TestAbiCli_CallOffChain(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to call off-chain contract methods.")

	rpc := PreTestRpc(t, RawUrl)

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
	t.Skip("Skipped by default. This test can be used to unpack contract methods.")

	abiCode := ``
	contract, err := abi.JSONToABIContract(strings.NewReader(abiCode))
	assert.NoError(t, err)

	data, err := base64.StdEncoding.DecodeString("")
	assert.NoError(t, err)

	id, err := contract.MethodById(data[0:4])
	assert.NoError(t, err)
	t.Log(id)
	var inputs types.Address

	err = contract.UnpackMethod(&inputs, id.Name, data)
	assert.NoError(t, err)

	t.Log(inputs)
}
