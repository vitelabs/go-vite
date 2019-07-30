package api

import (
	"encoding/base64"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	apidex "github.com/vitelabs/go-vite/rpcapi/api/dex"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm_db"
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

func (f DexTradeApi) GetOrderById(orderIdStr string) (*RpcOrder, error) {
	orderId, err := base64.StdEncoding.DecodeString(orderIdStr)
	if err != nil {
		return nil, err
	}
	if db, err := getDb(f.chain, types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		if matcher := dex.NewRawMatcher(db); err != nil {
			return nil, err
		} else {
			if order, err := matcher.GetOrderById(orderId); err != nil {
				return nil, err
			} else {
				return OrderToRpc(order), nil
			}
		}
	}
}

func (f DexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int) (ordersRes *OrdersRes, err error) {
	if fundDb, err := getDb(f.chain, types.AddressDexFund); err != nil {
		return nil, err
	} else {
		if marketInfo, ok := dex.GetMarketInfo(fundDb, tradeToken, quoteToken); !ok {
			return nil, dex.TradeMarketNotExistsErr
		} else {
			if tradeDb, err := getDb(f.chain, types.AddressDexTrade); err != nil {
				return nil, err
			} else {
				matcher := dex.NewMatcherWithMarketInfo(tradeDb, marketInfo)
				if ods, size, err := matcher.GetOrdersFromMarket(side, begin, end); err == nil {
					ordersRes = &OrdersRes{OrdersToRpc(ods), size}
					return ordersRes, err
				} else {
					return &OrdersRes{OrdersToRpc(ods), size}, err
				}
			}
		}
	}
}

func (f DexTradeApi) GetMarketInfoById(marketId int32) (ordersRes *apidex.RpcMarketInfo, err error) {
	if tradeDb, err := getDb(f.chain, types.AddressDexTrade); err != nil {
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
	if tradeDb, err := getDb(f.chain, types.AddressDexTrade); err != nil {
		return -1, err
	} else {
		return dex.GetTradeTimestamp(tradeDb), nil
	}
}

func getDb(c chain.Chain, address types.Address) (db vm_db.VmDb, err error) {
	prevHash, err := getPrevBlockHash(c, address)
	if err != nil {
		return nil, err
	}
	if db, err := vm_db.NewVmDb(c, &address, &c.GetLatestSnapshotBlock().Hash, prevHash); err != nil {
		return nil, err
	} else {
		return db, nil
	}
}

type RpcOrder struct {
	Id                   string   `json:"Id"`
	Address              string   `json:"Address"`
	MarketId             int32    `json:"MarketId"`
	Side                 bool     `json:"Side"`
	Type                 int32    `json:"Type"`
	Price                string   `json:"Price"`
	TakerFeeRate         int32    `json:"TakerFeeRate"`
	MakerFeeRate         int32    `json:"MakerFeeRate"`
	TakerBrokerFeeRate   int32    `json:"TakerBrokerFeeRate"`
	MakerBrokerFeeRate   int32    `json:"MakerBrokerFeeRate"`
	Quantity             string   `json:"Quantity"`
	Amount               string   `json:"Amount"`
	LockedBuyFee         string   `json:"LockedBuyFee,omitempty"`
	Status               int32    `json:"Status"`
	CancelReason         int32    `json:"CancelReason,omitempty"`
	ExecutedQuantity     string   `json:"ExecutedQuantity,omitempty"`
	ExecutedAmount       string   `json:"ExecutedAmount,omitempty"`
	ExecutedBaseFee      string   `json:"ExecutedBaseFee,omitempty"`
	ExecutedBrokerFee    string   `json:"ExecutedBrokerFee,omitempty"`
	RefundToken          string   `json:"RefundToken,omitempty"`
	RefundQuantity       string   `json:"RefundQuantity,omitempty"`
	Timestamp            int64    `json:"Timestamp"`
}

type OrdersRes struct {
	Orders []*RpcOrder `json:"orders,omitempty"`
	Size   int        `json:"size"`
}

func OrderToRpc(order *dex.Order) *RpcOrder {
	if order == nil {
		return nil
	}
	address, _ := types.BytesToAddress(order.Address)
	rpcOrder := &RpcOrder{}
	rpcOrder.Id = base64.StdEncoding.EncodeToString(order.Id)
	rpcOrder.Address = address.String()
	rpcOrder.MarketId = order.MarketId
	rpcOrder.Side = order.Side
	rpcOrder.Type = order.Type
	rpcOrder.Price = dex.BytesToPrice(order.Price)
	rpcOrder.TakerFeeRate = order.TakerFeeRate
	rpcOrder.MakerFeeRate = order.MakerFeeRate
	rpcOrder.TakerBrokerFeeRate = order.TakerBrokerFeeRate
	rpcOrder.MakerBrokerFeeRate = order.MakerBrokerFeeRate
	rpcOrder.Quantity = apidex.AmountBytesToString(order.Quantity)
	rpcOrder.Amount = apidex.AmountBytesToString(order.Amount)
	if len(order.LockedBuyFee) > 0 {
		rpcOrder.LockedBuyFee = apidex.AmountBytesToString(order.LockedBuyFee)
	}
	rpcOrder.Status = order.Status
	rpcOrder.CancelReason = order.CancelReason
	if len(order.ExecutedQuantity) > 0 {
		rpcOrder.ExecutedQuantity = apidex.AmountBytesToString(order.ExecutedQuantity)
	}
	if len(order.ExecutedAmount) > 0 {
		rpcOrder.ExecutedAmount = apidex.AmountBytesToString(order.ExecutedAmount)
	}
	if len(order.ExecutedBaseFee) > 0 {
		rpcOrder.ExecutedBaseFee = apidex.AmountBytesToString(order.ExecutedBaseFee)
	}
	if len(order.ExecutedBrokerFee) > 0 {
		rpcOrder.ExecutedBrokerFee = apidex.AmountBytesToString(order.ExecutedBrokerFee)
	}
	if len(order.RefundToken) > 0 {
		tk, _ := types.BytesToTokenTypeId(order.RefundToken)
		rpcOrder.RefundToken = tk.String()
	}
	if len(order.RefundQuantity) > 0 {
		rpcOrder.RefundQuantity = apidex.AmountBytesToString(order.RefundQuantity)
	}
	rpcOrder.Timestamp = order.Timestamp
	return rpcOrder
}

func OrdersToRpc(orders []*dex.Order) []*RpcOrder {
	if len(orders) == 0 {
		return nil
	} else {
		rpcOrders := make([]*RpcOrder, len(orders))
		for i := 0; i < len(orders); i++ {
			rpcOrders[i] = OrderToRpc(orders[i])
		}
		return rpcOrders
	}
}
