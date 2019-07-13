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
		{"type":"function","name":"DexTradeCleanExpireOrders", "inputs":[{"name":"data","type":"bytes"}]}
]`
	MethodNameDexTradeNewOrder    = "DexTradeNewOrder"
	MethodNameDexTradeCancelOrder = "DexTradeCancelOrder"
	MethodNameDexTradeNotifyNewMarket = "DexTradeNotifyNewMarket"
	MethodNameDexTradeCleanExpireOrders = "DexTradeCleanExpireOrders"
)

var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexTrade))
)
