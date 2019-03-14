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
	if matcher, err := f.getMatcher(); err != nil {
		return nil, err
	} else {
		makerBookId := dex.GetBookIdToMake(tradeToken.Bytes(), quoteToken.Bytes(), side)
		return matcher.GetOrderByIdAndBookId(orderId, makerBookId)
	}
}

func (f DexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int32) (order []*dex.Order, size int32, err error) {
	if matcher, err := f.getMatcher(); err != nil {
		return nil, 0, err
	} else {
		makerBookId := dex.GetBookIdToMake(tradeToken.Bytes(), quoteToken.Bytes(), side)
		return matcher.GetOrdersFromMarket(makerBookId, begin, end)
	}
}

func (f DexTradeApi) getMatcher() (matcher *dex.Matcher, err error) {
	if vmContext, err := vm_context.NewVmContext(f.chain, nil, nil, &types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		storage, _ := vmContext.(dex.BaseStorage)
		return dex.NewMatcher(&types.AddressDexTrade, &storage), nil
	}
}