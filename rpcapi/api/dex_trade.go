package api

import (
	"encoding/base64"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm_context"
)

type DexTradeApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewDexTradeApi(vite *vite.Vite) *DexTradeApi {
	return &DexTradeApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dextrade_api"),
	}
}

func (f DexTradeApi) String() string {
	return "DexTradeApi"
}


func (f DexTradeApi) GetOrderById(orderIdStr string, tradeToken, quoteToken types.TokenTypeId, side bool) (order *dex.Order, err error) {
	var (
		orderId []byte
	)
	orderId, err = base64.StdEncoding.DecodeString(orderIdStr)
	if err != nil {
		return nil, err
	}
	vmContext, _ := vm_context.NewVmContext(f.chain, nil, nil, &types.AddressDexTrade)
	storage, _ := vmContext.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	makerBookId := dex.GetBookIdToMake(tradeToken.Bytes(), quoteToken.Bytes(), side)
	return matcher.GetOrderByIdAndBookId(orderId, makerBookId)
}

func (f DexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, len int32) (order []*dex.Order, size int32, err error) {
	vmContext, _ := vm_context.NewVmContext(f.chain, nil, nil, &types.AddressDexTrade)
	storage, _ := vmContext.(dex.BaseStorage)
	matcher := dex.NewMatcher(&types.AddressDexTrade, &storage)
	makerBookId := dex.GetBookIdToMake(tradeToken.Bytes(), quoteToken.Bytes(), side)
	return matcher.PeekOrdersFromMarket(makerBookId, len)
}
