package api

import (
	"encoding/base64"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm_db"
)

type DexTradeApi struct {
	chain chain.Chain
	log   log15.Logger
}

type OrdersRes struct {
	Orders []*dex.Order `json:"Orders,omitempty"`
	Size   int32        `json:"Size"`
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
		return matcher.GetOrderByIdAndBookId(makerBookId, orderId)
	}
}

func (f DexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int32) (ordersRes *OrdersRes, err error) {
	if matcher, err := f.getMatcher(); err != nil {
		return nil, err
	} else {
		makerBookId := dex.GetBookIdToMake(tradeToken.Bytes(), quoteToken.Bytes(), side)
		if ods, size, err := matcher.GetOrdersFromMarket(makerBookId, begin, end); err == nil {
			ordersRes = &OrdersRes{ods, size}
			return ordersRes, err
		} else {
			return &OrdersRes{ods, size}, err
		}
	}
}

func (f DexTradeApi) TravelMarketOrders(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int32) (ordersRes *OrdersRes, err error) {
	if matcher, err := f.getMatcher(); err != nil {
		return nil, err
	} else {
		makerBookId := dex.GetBookIdToMake(tradeToken.Bytes(), quoteToken.Bytes(), side)
		if ods, size, err := matcher.GetOrdersFromMarket(makerBookId, begin, end); err == nil {
			ordersRes = &OrdersRes{ods, size}
			return ordersRes, err
		} else {
			return &OrdersRes{ods, size}, err
		}
	}
}

func (f DexTradeApi) getMatcher() (matcher *dex.Matcher, err error) {
	prevHash, err := getPrevBlockHash(f.chain, types.AddressDexTrade)
	if err != nil {
		return nil, err
	}
	if db, err := vm_db.NewVmDb(f.chain, &types.AddressDexTrade, &f.chain.GetLatestSnapshotBlock().Hash, prevHash); err != nil {
		return nil, err
	} else {
		storage, _ := db.(dex.BaseStorage)
		return dex.NewMatcher(&types.AddressDexTrade, &storage), nil
	}
}
