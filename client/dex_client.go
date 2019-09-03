package client

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
)

type DexClient interface {
	BuildRequestNewOrderBlock(param *dex.ParamPlaceOrder, selfAddr types.Address, prev *ledger.HashHeight) (block *api.AccountBlock, err error)
	BuildRequestCancelOrderBlock(param *dex.ParamDexCancelOrder, selfAddr types.Address, prev *ledger.HashHeight) (block *api.AccountBlock, err error)
}

func (c *client) BuildRequestNewOrderBlock(param *dex.ParamPlaceOrder, selfAddr types.Address, prev *ledger.HashHeight) (block *api.AccountBlock, err error) {
	data, err := buildDexNewOrderData(param)
	if err != nil {
		return nil, err
	}
	params := &RequestTxParams{}
	params.SelfAddr = selfAddr
	params.Data = data
	params.ToAddr = types.AddressDexFund
	params.Amount = big.NewInt(0)
	params.TokenId = ledger.ViteTokenId
	return c.BuildNormalRequestBlock(*params, prev)
}

func (c *client) BuildRequestCancelOrderBlock(param *dex.ParamDexCancelOrder, selfAddr types.Address, prev *ledger.HashHeight) (block *api.AccountBlock, err error) {
	data, err := buildDexCancelOrderData(param)
	if err != nil {
		return nil, err
	}
	params := &RequestTxParams{}
	params.SelfAddr = selfAddr
	params.Data = data
	params.ToAddr = types.AddressDexTrade
	params.Amount = big.NewInt(0)
	params.TokenId = ledger.ViteTokenId
	return c.BuildNormalRequestBlock(*params, prev)
}

func buildDexNewOrderData(param *dex.ParamPlaceOrder) ([]byte, error) {
	abiContract := abi.ABIDexFund
	methodName := abi.MethodNameDexFundNewOrder
	var arguments []interface{}
	arguments = append(arguments, param.TradeToken)
	arguments = append(arguments, param.QuoteToken)
	arguments = append(arguments, param.Side)
	arguments = append(arguments, param.OrderType)
	arguments = append(arguments, param.Price)
	arguments = append(arguments, param.Quantity)

	return abiContract.PackMethod(methodName, arguments...)
}

func buildDexCancelOrderData(param *dex.ParamDexCancelOrder) ([]byte, error) {
	abiContract := abi.ABIDexTrade
	methodName := abi.MethodNameDexTradeCancelOrder
	var arguments []interface{}
	arguments = append(arguments, param.OrderId)

	return abiContract.PackMethod(methodName, arguments...)
}
func parseDexNewOrderData(input []byte) (*dex.ParamPlaceOrder, error) {
	abiContract := abi.ABIDexFund
	methodName := abi.MethodNameDexFundNewOrder
	param := new(dex.ParamPlaceOrder)
	err := abiContract.UnpackMethod(param, methodName, input)
	return param, err
}

func parseDexCancelOrderData(input []byte) (*dex.ParamDexCancelOrder, error) {
	abiContract := abi.ABIDexTrade
	methodName := abi.MethodNameDexTradeCancelOrder
	param := new(dex.ParamDexCancelOrder)
	err := abiContract.UnpackMethod(param, methodName, input)
	return param, err
}
