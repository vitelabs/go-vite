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
	Size   int        `json:"Size"`
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
		db vm_db.VmDb
		matcher *dex.Matcher
	)
	orderId, err = base64.StdEncoding.DecodeString(orderIdStr)
	if err != nil {
		return nil, err
	}
	if db, err = f.getDb(types.AddressDexTrade); err != nil {
		return nil, err
	} else {
		if matcher = dex.NewRawMatcher(db); err != nil {
			return nil, err
		} else {
			return matcher.GetOrderById(orderId)
		}
	}
}

func (f DexTradeApi) GetOrdersFromMarket(tradeToken, quoteToken types.TokenTypeId, side bool, begin, end int) (ordersRes *OrdersRes, err error) {
	if fundDb, err := f.getDb(types.AddressDexFund); err != nil {
		return nil, err
	} else {
		if marketInfo, ok := dex.GetMarketInfo(fundDb, tradeToken, quoteToken); !ok {
			return nil, dex.TradeMarketNotExistsError
		} else {
			if tradeDb, err := f.getDb(types.AddressDexTrade); err != nil {
				return nil, err
			} else {
				matcher := dex.NewMatcherWithMarketInfo(tradeDb, marketInfo)
				if ods, size, err := matcher.GetOrdersFromMarket(side, begin, end); err == nil {
					ordersRes = &OrdersRes{ods, size}
					return ordersRes, err
				} else {
					return &OrdersRes{ods, size}, err
				}
			}
		}
	}
}

func (f DexTradeApi) getDb(address types.Address) (db vm_db.VmDb, err error) {
	prevHash, err := getPrevBlockHash(f.chain, address)
	if err != nil {
		return nil, err
	}
	if db, err := vm_db.NewVmDb(f.chain, &address, &f.chain.GetLatestSnapshotBlock().Hash, prevHash); err != nil {
		return nil, err
	} else {
		return db, nil
	}
}