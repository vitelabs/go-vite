package client

import (
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"gotest.tools/assert"
)

func TestClient_NewOrderInputs(t *testing.T) {
	abiContract := abi.ABIDexFund
	methodName := abi.MethodNameDexFundNewOrder

	method := abiContract.Methods[methodName]
	for k, v := range method.Inputs {
		t.Log(k, v.Name, v.Type, v.Indexed)
	}
	for k, v := range abi.ABIDexTrade.OffChains {
		t.Log(k, v)
	}
}

// wss://testnet.vitewallet.com/beta/ws
func TestClient_BuildRequestNewOrderBlock(t *testing.T) {
	viteTokeId, err := types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	assert.NilError(t, err)
	btcTokenId, err := types.HexToTokenTypeId("tti_322862b3f8edae3b02b110b1")
	assert.NilError(t, err)
	// 1000
	// 0.00001899

	price := "0.00001899"

	one := big.NewInt(1e18)
	quantity := big.NewInt(0).Set(one).Mul(one, big.NewInt(1000))
	data, err := buildDexNewOrderData(&dex.ParamPlaceOrder{
		TradeToken: viteTokeId,
		QuoteToken: btcTokenId,
		Side:       true,
		Price:      price,
		Quantity:   quantity,
	})
	assert.NilError(t, err)
	dataBase64 := base64.StdEncoding.EncodeToString(data)
	t.Log(dataBase64)

	expected := "FHkn7AAAAAAAAAAAAAAAAAAAAAAAAAAAAABWSVRFIFRPS0VOAAAAAAAAAAAAAAAAAAAAAAAAAAAAADIoYrP47a47ArEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADY1ya3F3qAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAKMC4wMDAwMTg5OQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	input, err := base64.StdEncoding.DecodeString(expected)
	assert.NilError(t, err)
	assert.Equal(t, expected, dataBase64)
	newOrder, err := parseDexNewOrderData(input)
	assert.NilError(t, err)
	t.Log(newOrder)

	//accountAddress: "vite_fea8b95db177c7bf8faf8826469c828731816e03149279cfd5"
	//amount: "0"
	//blockType: 2
	//data: "FHkn7AAAAAAAAAAAAAAAAAAAAAAAAAAAAABWSVRFIFRPS0VOAAAAAAAAAAAAAAAAAAAAAAAAAAAAADIoYrP47a47ArEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADY1ya3F3qAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAKMC4wMDAwMTg5OQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	//hash: "acacc7d94de1c752ca1164dca87a9e371002843734f04093b6d5456016af289a"
	//height: "12"
	//prevHash: "56ca8e95eb3ac0602ea20d3ffc0e72d8e3dda4c0154769c3d80793cfa3c5f409"
	//publicKey: "fbqh/FZDxv/vEmEF+JA2R34JKHLDqghT30WcyT1I5T0="
	//signature: "8lv/9STBd6zZEjQodf9NaGPOAYjpCNWFS7AU5gZYraqaSzTUOUY0o1NRyL1T7xSmhROy7ssjXSjpVWOWs4tRAQ=="
	//toAddress: "vite_0000000000000000000000000000000000000006e82b8ba657"
	//tokenId: "tti_5649544520544f4b454e6e40"

	for k, v := range abi.ABIDexFund.Events {
		t.Log(k, v)
	}

	for k, v := range abi.ABIDexTrade.Events {
		t.Log(k, v)
	}
}

func TestClient_BuildRequestCancelOrderBlock(t *testing.T) {
	orderId := "0000010100000000000001312d00005d2e049e000000"
	id, err := hex.DecodeString(orderId)
	assert.NilError(t, err)
	data, err := buildDexCancelOrderData(&dex.ParamDexCancelOrder{
		OrderId: id,
	})
	assert.NilError(t, err)

	dataBase64 := base64.StdEncoding.EncodeToString(data)
	t.Log(dataBase64)

	expected := "slGtxQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABYAAAEBAAAAAAAAATEtAABdLgSeAAAAAAAAAAAAAAAAAA=="
	input, err := base64.StdEncoding.DecodeString(expected)
	assert.NilError(t, err)
	assert.Equal(t, expected, dataBase64)
	newOrder, err := parseDexCancelOrderData(input)
	assert.NilError(t, err)
	t.Log(hex.EncodeToString(newOrder.OrderId))

	//accountAddress: "vite_fea8b95db177c7bf8faf8826469c828731816e03149279cfd5"
	//amount: "0"
	//blockType: 2
	//data: "slGtxQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABYAAAEBAAAAAAAAATEtAABdLgSeAAAAAAAAAAAAAAAAAA=="
	//hash: "1ad84a2af694bc2c93c7236d9188a4601ffcef744cebb87eccfd2ec4a3930dbe"
	//height: "14"
	//prevHash: "f277183f669358d7fb97a9a2000bc2429da812bae9be3e2b09b0bc63ad027229"
	//publicKey: "fbqh/FZDxv/vEmEF+JA2R34JKHLDqghT30WcyT1I5T0="
	//signature: "n2ERe/6V8IV0Zi4J6qPAeG0u+V+rYeZ4S0W+m2x+tglKRri3Z20oHXvkxBD7D7GQsJR7m1/OxUU2gHY6vOHjCg=="
	//toAddress: "vite_00000000000000000000000000000000000000079710f19dc7"
	//tokenId: "tti_5649544520544f4b454e6e40"
}
