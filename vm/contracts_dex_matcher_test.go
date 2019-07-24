package vm

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
	"testing"
	"time"
)

type tokenInfo struct {
	tokenId  types.TokenTypeId
	Symbol   string
	decimals int32
	Index    int8
}

var (
	ETH  = tokenInfo{types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}, "ETH", 12, 0}  //tradeToken
	VITE = tokenInfo{types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}, "VITE", 14, 0} //quoteToken
	db   *testDatabase
)

func TestDexMatcher(t *testing.T) {
	marketId := initDexFundDbAndMarket()
	innerTestMatcher(t, marketId)
	innerTestFeeCalculation(t)
	innerTestDustCheck(t)
	innerTestDustWithOrder(t)
}

func innerTestMatcher(t *testing.T, marketId int32) {
	mc, err := dex.NewMatcher(db, marketId)
	assert.True(t, err == nil)

	dex.SetFeeRate("0.06", "0.05") // takerFee, makerFee
	// buy
	buy1, buy1Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100.02", 1000)
	buy2, buy2Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100.03", 3000)
	buy3, buy3Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100.02", 7000)
	buy4, buy4Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100.04", 4500)
	buy5, buy5Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100.01", 2000)
	mc.MatchOrder(buy1)
	mc.MatchOrder(buy2)
	mc.MatchOrder(buy3)
	mc.MatchOrder(buy4)
	mc.MatchOrder(buy5)
	assert.Equal(t, 5, len(db.logList))

	// buy4[100.04] -> buy2[100.03] -> buy1[100.02] -> buy3[100.02] -> buy5[100.01]

	buyOrders, buySize, err := mc.GetOrdersFromMarket(false, 0, 10)
	assert.Equal(t, 5, buySize)
	assert.Equal(t, buy4Id, buyOrders[0].Id)
	assert.Equal(t, buy2Id, buyOrders[1].Id)
	assert.Equal(t, buy1Id, buyOrders[2].Id)
	assert.Equal(t, buy3Id, buyOrders[3].Id)
	assert.Equal(t, buy5Id, buyOrders[4].Id)

	sell1, sell1Id := newOrderInfo(ETH.tokenId, VITE.tokenId, true, dex.Limited, "100.1", 10000)
	sell2, sell2Id := newOrderInfo(ETH.tokenId, VITE.tokenId, true, dex.Limited, "100.02", 5000)
	mc.MatchOrder(sell1)
	assert.Equal(t, 6, len(db.logList))
	mc.MatchOrder(sell2)

	// sell1Id[100.1]
	sellOrders, sellSize, err := mc.GetOrdersFromMarket(true, 0, 10)
	assert.Equal(t, 1, sellSize)
	assert.Equal(t, sell1Id, sellOrders[0].Id)

	// buy2[100.03] -> buy1100.02] -> buy3[100.02] -> buy5[100.01]
	buyOrders, buySize, err = mc.GetOrdersFromMarket(false, 0, 10)

	assert.Equal(t, 4, buySize)
	assert.Equal(t, buy2Id, buyOrders[0].Id)

	// buy2[100.03]
	od, err := mc.GetOrderById(buy2Id)
	assert.True(t, err == nil)
	assert.True(t, od.Status == dex.PartialExecuted)
	assert.True(t, big.NewInt(500).Cmp(new(big.Int).SetBytes(od.ExecutedQuantity)) == 0)
	assert.True(t, big.NewInt(250075).Cmp(new(big.Int).SetBytes(od.ExecutedFee)) == 0) // 500 * 100.03 * 100 * 0.05

	// db.logList[0] orderEvent(buy1) newMaker
	// db.logList[1] orderEvent(buy2) newMaker
	// db.logList[2] orderEvent(buy3) newMaker
	// db.logList[3] orderEvent(buy4) newMaker
	// db.logList[4] orderEvent(buy5) newMaker
	// db.logList[5] orderEvent(sell1) newMaker
	// db.logList[6] orderEvent(sell2) newTaker
	// db.logList[7] orderEvent(buy4) makerFilled
	// db.logList[8] orderEvent(buy2) makerPartialFilled
	// db.logList[9] txEvent taker -> maker[sell2, buy4]
	// db.logList[10] txEvent taker -> maker[sell2, buy2]
	assert.Equal(t, 11, len(db.logList))

	_, err = mc.GetOrderById(buy4Id)
	assert.Equal(t, err, dex.OrderNotExistsErr)

	log := db.logList[6]
	newOrderEvent := dex.NewOrderEvent{}
	newOrderEvent = newOrderEvent.FromBytes(log.Data).(dex.NewOrderEvent)
	assert.Equal(t, sell2Id, newOrderEvent.Order.Id)
	assert.Equal(t, dex.FullyExecuted, int(newOrderEvent.Order.Status))
	assert.True(t, checkBigEqualToInt(5000, newOrderEvent.Order.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(50019500, newOrderEvent.Order.ExecutedAmount)) // (4500*100.04 + 500 * 100.03) * 100
	assert.True(t, checkBigEqualToInt(3001170, newOrderEvent.Order.ExecutedFee))     // 45018000 * 0.06 + 5001500 * 0.06

	log = db.logList[7]
	odEvent := dex.OrderUpdateEvent{}
	odEvent = odEvent.FromBytes(log.Data).(dex.OrderUpdateEvent)
	assert.Equal(t, buy4Id, odEvent.Id)
	assert.Equal(t, dex.FullyExecuted, int(odEvent.Status))
	assert.True(t, checkBigEqualToInt(4500, odEvent.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(45018000, odEvent.ExecutedAmount)) // 4500*100.04 * 100
	assert.True(t, checkBigEqualToInt(2250900, odEvent.ExecutedFee))     // 45018000 * 0.05

	log = db.logList[8]
	odEvent = dex.OrderUpdateEvent{}
	odEvent = odEvent.FromBytes(log.Data).(dex.OrderUpdateEvent)
	assert.Equal(t, buy2Id, odEvent.Id)
	assert.Equal(t, dex.PartialExecuted, int(odEvent.Status))
	assert.True(t, checkBigEqualToInt(500, odEvent.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(5001500, odEvent.ExecutedAmount)) // 500 * 100.03 * 100
	assert.True(t, checkBigEqualToInt(250075, odEvent.ExecutedFee))     // 5001500 * 0.05

	log = db.logList[9]
	txEvent := dex.TransactionEvent{}
	txEvent = txEvent.FromBytes(log.Data).(dex.TransactionEvent)
	assert.Equal(t, sell2Id, txEvent.TakerId)
	assert.Equal(t, buy4Id, txEvent.MakerId)
	assert.True(t, checkBigEqualToInt(4500, txEvent.Quantity))
	assert.True(t, priceEqual("100.04", txEvent.Price))
	assert.True(t, checkBigEqualToInt(45018000, txEvent.Amount))  // 4500*100.04 * 100
	assert.True(t, checkBigEqualToInt(2701080, txEvent.TakerFee)) // 45018000 * 0.06
	assert.True(t, checkBigEqualToInt(2250900, txEvent.MakerFee)) // 45018000 * 0.05

	log = db.logList[10]
	txEvent = dex.TransactionEvent{}
	txEvent = txEvent.FromBytes(log.Data).(dex.TransactionEvent)
	assert.Equal(t, sell2Id, txEvent.TakerId)
	assert.Equal(t, buy2Id, txEvent.MakerId)
	assert.True(t, checkBigEqualToInt(500, txEvent.Quantity))
	assert.True(t, priceEqual("100.03", txEvent.Price))
	assert.True(t, checkBigEqualToInt(5001500, txEvent.Amount))
	assert.True(t, checkBigEqualToInt(300090, txEvent.TakerFee))
	assert.True(t, checkBigEqualToInt(250075, txEvent.MakerFee))

	buy6, buy6Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100.3", 10100)
	mc.MatchOrder(buy6)

	buyOrders, buySize, err = mc.GetOrdersFromMarket(false, 0, 10)
	sellOrders, sellSize, err = mc.GetOrdersFromMarket(true, 0, 10)

	assert.Equal(t, buy6Id, buyOrders[0].Id)
	assert.Equal(t, 0, sellSize)
	assert.Equal(t, 5, buySize)

	assert.Equal(t, 14, len(db.logList))

	// db.logList[11] orderEvent(buy6Id) newTaker 10100
	// db.logList[12] orderEvent(sell1Id) makerFilled 10000
	// db.logList[13] txEvent taker -> maker[buy6Id, sell1Id]
	log = db.logList[11]
	newOrderEvent = dex.NewOrderEvent{}
	newOrderEvent = newOrderEvent.FromBytes(log.Data).(dex.NewOrderEvent)
	assert.Equal(t, buy6Id, newOrderEvent.Order.Id)
	assert.Equal(t, dex.PartialExecuted, int(newOrderEvent.Order.Status))
	assert.True(t, checkBigEqualToInt(10000, newOrderEvent.Order.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(100100000, newOrderEvent.Order.ExecutedAmount)) // 10000 * 100.1 * 100
	assert.True(t, checkBigEqualToInt(6006000, newOrderEvent.Order.ExecutedFee))      // 100100000 * 0.06

	log = db.logList[12]
	odEvent = dex.OrderUpdateEvent{}
	odEvent = odEvent.FromBytes(log.Data).(dex.OrderUpdateEvent)
	assert.Equal(t, sell1Id, odEvent.Id)
	assert.Equal(t, dex.FullyExecuted, int(odEvent.Status))
	assert.True(t, checkBigEqualToInt(10000, odEvent.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(100100000, odEvent.ExecutedAmount))
	assert.True(t, checkBigEqualToInt(5005000, odEvent.ExecutedFee)) // 100100000 * 0.05

	log = db.logList[13]
	txEvent = dex.TransactionEvent{}
	txEvent = txEvent.FromBytes(log.Data).(dex.TransactionEvent)
	assert.Equal(t, buy6Id, txEvent.TakerId)
	assert.Equal(t, sell1Id, txEvent.MakerId)
	assert.True(t, checkBigEqualToInt(10000, txEvent.Quantity))
	assert.True(t, priceEqual("100.1", txEvent.Price))
	assert.True(t, checkBigEqualToInt(100100000, txEvent.Amount))
	assert.True(t, checkBigEqualToInt(6006000, txEvent.TakerFee))
	assert.True(t, checkBigEqualToInt(5005000, txEvent.MakerFee))
}

func innerTestFeeCalculation(t *testing.T) {
	dex.SetFeeRate("0.07", "0.05") // takerFee, makerFee
	buyTakerOrder, _ := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "0.001234211", 1000000)
	assert.True(t, checkBigEqualToInt(8639, buyTakerOrder.Order.LockedBuyFee)) // 123421 * 0.07

	buyTakerOrder1, _ := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "0.001234231", 1000000)
	assert.True(t, checkBigEqualToInt(8640, buyTakerOrder1.Order.LockedBuyFee)) //123423 * 0.07
	buyTakerOrder1.Order.ExecutedFee = new(big.Int).SetInt64(8634).Bytes()      // 8640 - 8634 = 6

	feeBytes, executedFee := dex.CalculateFeeAndExecutedFee(buyTakerOrder1, new(big.Int).SetInt64(100).Bytes(), "0.07")
	assert.True(t, checkBigEqualToInt(6, feeBytes))
	assert.True(t, checkBigEqualToInt(8640, executedFee))
}

func innerTestDustCheck(t *testing.T) {
	order := &dex.Order{}
	order.Quantity = big.NewInt(2000).Bytes()
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.Price = dex.PriceToBytes("0.1")
	assert.False(t, dex.IsDust(order, big.NewInt(1000).Bytes(), 2))
	assert.True(t, dex.IsDust(order, big.NewInt(1001).Bytes(), 2))

	order1 := &dex.Order{}
	order1.Quantity = big.NewInt(1000).Bytes()
	order1.ExecutedQuantity = big.NewInt(0).Bytes()
	order1.Price = dex.PriceToBytes("0.001")
	assert.False(t, dex.IsDust(order1, big.NewInt(990).Bytes(), -2))
	assert.True(t, dex.IsDust(order1, big.NewInt(991).Bytes(), -2))
}

func innerTestDustWithOrder(t *testing.T) {
	marketId := initDexFundDbAndMarket()
	mc, err := dex.NewMatcher(db, marketId)

	assert.True(t, err == nil)

	dex.SetFeeRate("0.06", "0.05") // takerFee, makerFee
	// buy quantity = origin * 100,000,000
	buy1, buy1Id := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "0.00012345", 10000) //amount 123.45
	mc.MatchOrder(buy1)

	buy1New, err := mc.GetOrderById(buy1Id)
	assert.True(t, err == nil)
	assert.True(t, checkBigEqualToInt(7, buy1New.Order.LockedBuyFee))

	// sell
	sell1, sell1Id := newOrderInfo(ETH.tokenId, VITE.tokenId, true, dex.Limited, "0.00012342", 10002) // amount 123.44
	mc.MatchOrder(sell1)
	// sell order.Quantity 10002 order.ExecutedQuantity 10000
	// roundAmount((10002-10000) * 0.00012345 * 100) = 0

	_, buySize, err := mc.GetOrdersFromMarket(false, 0, 10)
	_, sellSize, err := mc.GetOrdersFromMarket(false, 0, 10)

	assert.Equal(t, 0, buySize)
	assert.Equal(t, 0, sellSize)

	// db.logList[0] orderEvent(301) newMaker
	// db.logList[1] orderEvent(401) newTaker
	// db.logList[2] orderEvent(301) makerFilled
	// db.logList[3] txEvent taker -> maker[buy6Id, sell1Id]
	assert.Equal(t, 4, len(db.logList))
	log := db.logList[1]
	newOrderEvent := dex.NewOrderEvent{} //taker
	newOrderEvent = newOrderEvent.FromBytes(log.Data).(dex.NewOrderEvent)
	assert.Equal(t, sell1Id, newOrderEvent.Order.Id)
	assert.Equal(t, dex.FullyExecuted, int(newOrderEvent.Order.Status))
	assert.True(t, checkBigEqualToInt(10000, newOrderEvent.Order.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(123, newOrderEvent.Order.ExecutedAmount))
	assert.True(t, checkBigEqualToInt(7, newOrderEvent.Order.ExecutedFee))
	assert.Equal(t, ETH.tokenId.Bytes(), newOrderEvent.Order.RefundToken)
	assert.True(t, checkBigEqualToInt(2, newOrderEvent.Order.RefundQuantity))

	log = db.logList[2]
	orderEvent := dex.OrderUpdateEvent{} // maker
	orderEvent = orderEvent.FromBytes(log.Data).(dex.OrderUpdateEvent)
	assert.Equal(t, buy1Id, orderEvent.Id)
	assert.Equal(t, dex.FullyExecuted, int(orderEvent.Status))
	assert.True(t, checkBigEqualToInt(10000, orderEvent.ExecutedQuantity))
	assert.True(t, checkBigEqualToInt(123, orderEvent.ExecutedAmount))
	assert.True(t, checkBigEqualToInt(6, orderEvent.ExecutedFee))
	assert.Equal(t, VITE.tokenId.Bytes(), orderEvent.RefundToken)
	assert.True(t, checkBigEqualToInt(1, orderEvent.RefundQuantity)) // LockedBuyFee - ExecutedFee

	log = db.logList[3]
	txEvent := dex.TransactionEvent{} // maker
	txEvent = txEvent.FromBytes(log.Data).(dex.TransactionEvent)
	assert.Equal(t, true, txEvent.TakerSide)
	assert.Equal(t, sell1Id, txEvent.TakerId)
	assert.Equal(t, buy1Id, txEvent.MakerId)
	assert.True(t, priceEqual("0.00012345", txEvent.Price))
	assert.True(t, checkBigEqualToInt(10000, txEvent.Quantity))
	assert.True(t, checkBigEqualToInt(123, txEvent.Amount))
	assert.True(t, checkBigEqualToInt(7, txEvent.TakerFee))
	assert.True(t, checkBigEqualToInt(6, txEvent.MakerFee))
}

func checkBigEqualToInt(expected int, value []byte) bool {
	return new(big.Int).SetUint64(uint64(expected)).Cmp(new(big.Int).SetBytes(value)) == 0
}

func initDexFundDbAndMarket() int32 {
	var marketId int32 = 1
	db = NewNoDatabase()
	db.addr = types.AddressDexTrade
	initTimerTimestamp()
	dex.SaveMarketInfo(db, getMarketInfo(marketId, ETH, VITE), ETH.tokenId, VITE.tokenId) //should not be store in trade storage, only for newOrder render
	dex.SaveMarketInfoById(db, getMarketInfo(marketId, ETH, VITE))
	return marketId
}

func initTimerTimestamp() {
	dex.SetTimerTimestamp(db, time.Now().Unix())
}

func getMarketInfo(marketId int32, tradeToken, quoteToken tokenInfo) *dex.MarketInfo {
	marketInfo := &dex.MarketInfo{}
	marketInfo.MarketId = marketId
	marketInfo.MarketSymbol = fmt.Sprintf("%s-%d_%s-%d", tradeToken.Symbol, tradeToken.Index, quoteToken.Symbol, quoteToken.Index)
	marketInfo.TradeToken = tradeToken.tokenId.Bytes()
	marketInfo.QuoteToken = quoteToken.tokenId.Bytes()
	marketInfo.TradeTokenDecimals = tradeToken.decimals
	marketInfo.QuoteTokenDecimals = quoteToken.decimals
	marketInfo.AllowMine = true
	marketInfo.Valid = true
	marketInfo.Creator = []byte("123456789012345678901")
	marketInfo.Timestamp = time.Now().Unix()
	return marketInfo
}

func newOrderInfo(tradeToken, quoteToken types.TokenTypeId, side bool, orderType int32, price string, quantity uint64) (*dex.Order, []byte) {
	param := &dex.ParamDexFundNewOrder{}
	param.TradeToken = tradeToken
	param.QuoteToken = quoteToken
	param.Side = side
	param.OrderType = int8(orderType)
	param.Price = price
	param.Quantity = new(big.Int).SetUint64(quantity)
	order := &dex.Order{}
	address := types.Address{}
	address.SetBytes([]byte("123456789012345678901"))
	dex.RenderOrder(order, param, db, address)
	return order, order.Id
}

func priceEqual(price string, bs []byte) bool {
	return bytes.Equal(dex.PriceToBytes(price), bs)
}
