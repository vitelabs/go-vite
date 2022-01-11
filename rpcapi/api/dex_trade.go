package api

import (
	"encoding/hex"
	"fmt"

	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/log15"
	apidex "github.com/vitelabs/go-vite/v2/rpcapi/api/dex"
	"github.com/vitelabs/go-vite/v2/vm/contracts/dex"
)

type DexTradeApi struct {
	vite  *vite.Vite
	chain chain.Chain
	log   log15.Logger
}

func NewDexTradeApi(vite *vite.Vite) *DexTradeApi {
	return &DexTradeApi{
		vite:  vite,
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dextrade_api"),
	}
}

func (f DexTradeApi) String() string {
	return "DexTradeApi"
}

func (f DexTradeApi) GetOrderById(orderIdStr string) (*apidex.RpcOrder, error) {
	orderId, err := hex.DecodeString(orderIdStr)
	if err != nil {
		return nil, err
	}
	if db, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		return apidex.InnerGetOrderById(db, orderId)
	}
}

func (f DexTradeApi) GetOrderBySendHash(sendHash types.Hash) (*apidex.RpcOrder, error) {
	if db, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		if orderId, ok := dex.GetOrderIdByHash(db, sendHash.Bytes()); !ok {
			return nil, dex.OrderNotExistsErr
		} else {
			return apidex.InnerGetOrderById(db, orderId)
		}
	}
}

func (f DexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int) (ordersRes *apidex.OrdersRes, err error) {
	if end-begin > 10000 {
		return nil, fmt.Errorf("end - begin must be less than 10000")
	}
	if fundDb, err := getVmDb(f.chain, types.AddressDexFund); err != nil {
		return nil, err
	} else {
		latest, err := f.chain.GetLatestAccountBlock(types.AddressDexTrade)
		if err != nil {
			return nil, err
		}
		if marketInfo, ok := dex.GetMarketInfo(fundDb, tradeToken, quoteToken); !ok {
			return nil, dex.TradeMarketNotExistsErr
		} else {
			if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
				return nil, err
			} else {
				matcher := dex.NewMatcherWithMarketInfo(tradeDb, marketInfo)
				ods, size, err := matcher.GetOrdersFromMarket(side, begin, end)
				latest2, _ := f.chain.GetLatestAccountBlock(types.AddressDexTrade)
				return &apidex.OrdersRes{apidex.OrdersToRpc(ods), size, latest.HashHeight(), latest2.HashHeight()}, err
			}
		}
	}
}

type MarketOrderParam struct {
	TradeToken types.TokenTypeId
	QuoteToken types.TokenTypeId
	SellBegin  int
	SellEnd    int
	BuyBegin   int
	BuyEnd     int
}

func (f DexTradeApi) GetMarketOrders(param MarketOrderParam) (ordersRes *apidex.OrdersRes, err error) {
	if param.SellEnd-param.SellBegin > 10000 {
		return nil, fmt.Errorf("sell end - begin must be less than 10000")
	}
	if param.BuyEnd-param.BuyBegin > 10000 {
		return nil, fmt.Errorf("sell end - begin must be less than 10000")
	}
	if fundDb, err := getVmDb(f.chain, types.AddressDexFund); err != nil {
		return nil, err
	} else {
		latest, err := f.chain.GetLatestAccountBlock(types.AddressDexTrade)
		if err != nil {
			return nil, err
		}
		if marketInfo, ok := dex.GetMarketInfo(fundDb, param.TradeToken, param.QuoteToken); !ok {
			return nil, dex.TradeMarketNotExistsErr
		} else {
			if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
				return nil, err
			} else {
				var obs []*apidex.RpcOrder
				matcher := dex.NewMatcherWithMarketInfo(tradeDb, marketInfo)
				if param.SellEnd > param.SellBegin {
					sellOds, _, err := matcher.GetOrdersFromMarket(true, param.SellBegin, param.SellEnd)
					if err != nil {
						return nil, err
					}
					obs = append(obs, apidex.OrdersToRpc(sellOds)...)
				}

				if param.BuyEnd > param.BuyBegin {
					buyOds, _, err := matcher.GetOrdersFromMarket(false, param.BuyBegin, param.BuyEnd)
					if err != nil {
						return nil, err
					}
					obs = append(obs, apidex.OrdersToRpc(buyOds)...)
				}
				latest2, err := f.chain.GetLatestAccountBlock(types.AddressDexTrade)
				if err != nil {
					return nil, err
				}
				return &apidex.OrdersRes{obs, len(obs), latest.HashHeight(), latest2.HashHeight()}, nil
			}
		}
	}
}

func (f DexTradeApi) GetMarketInfoById(marketId int32) (ordersRes *apidex.RpcMarketInfo, err error) {
	if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		if marketInfo, ok := dex.GetMarketInfoById(tradeDb, marketId); ok {
			return apidex.MarketInfoToRpc(marketInfo), nil
		} else {
			return nil, nil
		}
	}
}

func (f DexTradeApi) GetTimestamp() (timestamp int64, err error) {
	if tradeDb, err := getVmDb(f.chain, types.AddressDexTrade); err != nil {
		return -1, err
	} else {
		return dex.GetTradeTimestamp(tradeDb), nil
	}
}
