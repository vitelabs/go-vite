package rpc

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api/dex"
)

// ContractApi ...
type DexTradeApi interface {
	GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int) (ordersRes *dex.OrdersRes, err error)
}

type dexTradeApi struct {
	cc *rpc.Client
}

func NewDexTradeApi(cc *rpc.Client) DexTradeApi {
	return &dexTradeApi{cc: cc}
}

func (ci dexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int) (result *dex.OrdersRes, err error) {
	result = &dex.OrdersRes{}
	err = ci.cc.Call(&result, "dextrade_getOrdersFromMarket", tradeToken, quoteToken, side, begin, end)
	return
}
