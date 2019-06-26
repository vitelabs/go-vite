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

func NewDexTradeApi(vite *vite.Vite) *DexTradeApi {
	return &DexTradeApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/dextrade_api"),
	}
}

func (f DexTradeApi) String() string {
	return "DexTradeApi"
}

func (f DexTradeApi) GetOrderById(orderIdStr string, tradeToken, quoteToken types.TokenTypeId, side bool) (*RpcOrder, error) {

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

}

type OrdersRes struct {
	Orders []*RpcOrder `json:"orders,omitempty"`
	Size   int        `json:"size"`
}

func OrderToRpc(order *dex.Order) *RpcOrder {
	if order == nil {
		return nil
	}
	rpcOrder := &RpcOrder{}
	return rpcOrder
}

func OrdersToRpc(orders []*dex.Order) []*RpcOrder {
	if len(orders) == 0 {
		return nil
	} else {
		rpcOrders := make([]*RpcOrder, len(orders))
		return rpcOrders
	}
}
