package abi

import (
	"github.com/vitelabs/go-vite/vm/abi"
	"strings"
)

const (
	jsonDexTrade = `
	[
		{"type":"function","name":"DexTradeNewOrder", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexTradeCancelOrder", "inputs":[{"name":"orderId","type":"bytes"}]},
		{"type":"function","name":"DexTradeNotifyNewMarket", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexTradeCleanExpireOrders", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexTradeCancelOrderByHash", "inputs":[{"name":"sendHash","type":"bytes32"}]},

		{"type":"function","name":"PlaceOrder", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"CancelOrder", "inputs":[{"name":"orderId","type":"bytes"}]},
		{"type":"function","name":"SyncNewMarket", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"ClearExpiredOrders", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"CancelOrderByTransactionHash", "inputs":[{"name":"sendHash","type":"bytes32"}]}

]`
	MethodNameDexTradeNewOrder          = "DexTradeNewOrder"
	MethodNameDexTradeCancelOrder       = "DexTradeCancelOrder"
	MethodNameDexTradeNotifyNewMarket   = "DexTradeNotifyNewMarket"
	MethodNameDexTradeCleanExpireOrders = "DexTradeCleanExpireOrders"
	MethodNameDexTradeCancelOrderByHash = "DexTradeCancelOrderByHash"

	MethodNameDexTradePlaceOrder                   = "PlaceOrder"
	MethodNameDexTradeCancelOrderV2                = "CancelOrder"
	MethodNameDexTradeSyncNewMarket                = "SyncNewMarket"
	MethodNameDexTradeClearExpiredOrders           = "ClearExpiredOrders"
	MethodNameDexTradeCancelOrderByTransactionHash = "CancelOrderByTransactionHash"
)

var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexTrade))
)
