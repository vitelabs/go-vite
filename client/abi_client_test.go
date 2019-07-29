package client

import (
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestAbiCli_CallOffChain(t *testing.T) {
	rpc, err := NewRpcClient(RawUrl)
	if err != nil {
		t.Error(err)
		return
	}

	abi := `xxx`
	offchainCode := `xxx`
	contract := types.HexToAddressPanic("xxxx")

	abiCli, err := GetAbiCli(rpc, abi, offchainCode, contract)
	if err != nil {
		t.Fatal(err)
	}

	result, err := abiCli.CallOffChain("xxx")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range result {
		t.Log(k, v)
	}
}
