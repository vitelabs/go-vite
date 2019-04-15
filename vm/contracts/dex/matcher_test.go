package dex

import (
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"testing"
	"time"
)

type tokenInfo struct {
	tokenId types.TokenTypeId
	decimals int32
}

var (
	ETH = tokenInfo{types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}, 12} //tradeToken
	VITE = tokenInfo{types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}, 14} //quoteToken
)

func TestMatcher(t *testing.T) {
	localStorage := NewMapStorage()
	st := BaseStorage(&localStorage)
	mc := NewMatcher(getAddress(), &st)

	DeleteTerminatedOrder = true
	SetFeeRate("0.06", "0.05") // takerFee, makerFee
	// buy
	buy1 := newOrderInfo(101, ETH, VITE, false, Limited, "100.02", 1000, time.Now().UnixNano()/1000)
	buy2 := newOrderInfo(102, ETH, VITE, false, Limited, "100.03", 3000, time.Now().UnixNano()/1000)
	buy3 := newOrderInfo(103, ETH, VITE, false, Limited, "100.02", 7000, time.Now().UnixNano()/1000)
	buy4 := newOrderInfo(104, ETH, VITE, false, Limited, "100.04", 4500, time.Now().UnixNano()/1000)
	buy5 := newOrderInfo(105, ETH, VITE, false, Limited, "100.01", 2000, time.Now().UnixNano()/1000)
	mc.MatchOrder(buy1)
	mc.MatchOrder(buy2)
	mc.MatchOrder(buy3)
	mc.MatchOrder(buy4)
	mc.MatchOrder(buy5)
	assert.Equal(t, 5, len(localStorage.logs))

	// 104[100.04] -> 102[100.03] -> 101[100.02] -> 103[100.02] -> 105[100.01]
	bookNameToMakeForBuy := getBookIdToMakeForTaker(buy5)
	assert.Equal(t,104, fromOrderIdToInt(mc.books[bookNameToMakeForBuy].header))

	orders, size, err := mc.GetOrdersFromMarket(bookNameToMakeForBuy, 0, 10)
	assert.Equal(t, 5, len(orders))
	assert.Equal(t, int32(5), size)
	assert.Equal(t, 104, fromOrderIdBytesToInt(orders[0].Id))
	assert.Equal(t, 102, fromOrderIdBytesToInt(orders[1].Id))
	assert.Equal(t, 101, fromOrderIdBytesToInt(orders[2].Id))
	assert.Equal(t, 103, fromOrderIdBytesToInt(orders[3].Id))
	assert.Equal(t, 105, fromOrderIdBytesToInt(orders[4].Id))

	sell1 := newOrderInfo(201, ETH, VITE, true, Limited, "100.1", 10000, time.Now().UnixNano()/1000)
	sell2 := newOrderInfo(202, ETH, VITE, true, Limited, "100.02", 5000, time.Now().UnixNano()/1000)
	mc.MatchOrder(sell1)
	assert.Equal(t, 6, len(localStorage.logs))
	mc.MatchOrder(sell2)

	// 201[100.1]
	bookIdToMakeForSell := getBookIdToMakeForTaker(sell1)
	assert.Equal(t,int32(1), mc.books[bookIdToMakeForSell].length)
	assert.Equal(t,201, fromOrderIdToInt(mc.books[bookIdToMakeForSell].header))

	// 102[100.03] -> 101[100.02] -> 103[100.02] -> 105[100.01]
	assert.Equal(t,int32(4), mc.books[bookNameToMakeForBuy].length)
	assert.Equal(t,102, fromOrderIdToInt(mc.books[bookNameToMakeForBuy].header))

	// 102[100.03]
	pl, _, _, _  := mc.books[bookIdToMakeForSell].getByKey(mc.books[bookNameToMakeForBuy].header)
	od, _ := (*pl).(Order)
	assert.True(t, od.Status == PartialExecuted)
	assert.True(t, big.NewInt(500).Cmp(new(big.Int).SetBytes(od.ExecutedQuantity)) == 0)
	assert.True(t, big.NewInt(250075).Cmp(new(big.Int).SetBytes(od.ExecutedFee)) == 0) // 500 * 100.03 * 100 * 0.05

	// localStorage.logs[0] orderEvent(101) newMaker
	// localStorage.logs[1] orderEvent(102) newMaker
	// localStorage.logs[2] orderEvent(103) newMaker
	// localStorage.logs[3] orderEvent(104) newMaker
	// localStorage.logs[4] orderEvent(105) newMaker
	// localStorage.logs[5] orderEvent(201) newMaker
	// localStorage.logs[6] orderEvent(202) newTaker
	// localStorage.logs[7] orderEvent(104) makerFilled
	// localStorage.logs[8] orderEvent(102) makerPartialFilled
	// localStorage.logs[9] txEvent taker -> maker[202, 104]
	// localStorage.logs[10] txEvent taker -> maker[202, 102]
	assert.Equal(t, 11, len(localStorage.logs))

	buy4OrderId, _ := NewOrderId(orderIdBytesFromInt(104))
	pl, _, _ , err = mc.books[bookIdToMakeForSell].getByKey(buy4OrderId)
	assert.True(t, err != nil)
	assert.True(t, pl == nil)

	log := localStorage.logs[6]
	newOrderEvent := NewOrderEvent{}
	newOrderEvent = newOrderEvent.FromBytes(log.Data).(NewOrderEvent)
	assert.Equal(t, 202, fromOrderIdBytesToInt(newOrderEvent.Order.Id))
	assert.Equal(t, FullyExecuted, int(newOrderEvent.Order.Status))
	assert.True(t, CheckBigEqualToInt(5000, newOrderEvent.Order.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(50019500, newOrderEvent.Order.ExecutedAmount)) // (4500*100.04 + 500 * 100.03) * 100
	assert.True(t, CheckBigEqualToInt(3001170, newOrderEvent.Order.ExecutedFee)) // 45018000 * 0.06 + 5001500 * 0.06

	log = localStorage.logs[7]
	odEvent := OrderUpdateEvent{}
	odEvent = odEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 104, fromOrderIdBytesToInt(odEvent.Id))
	assert.Equal(t, FullyExecuted, int(odEvent.Status))
	assert.True(t, CheckBigEqualToInt(4500, odEvent.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(45018000, odEvent.ExecutedAmount)) // 4500*100.04 * 100
	assert.True(t, CheckBigEqualToInt(2250900, odEvent.ExecutedFee)) // 45018000 * 0.05

	log = localStorage.logs[8]
	odEvent = OrderUpdateEvent{}
	odEvent = odEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 102, fromOrderIdBytesToInt(odEvent.Id))
	assert.Equal(t, PartialExecuted, int(odEvent.Status))
	assert.True(t, CheckBigEqualToInt(500, odEvent.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(5001500, odEvent.ExecutedAmount)) // 500 * 100.03 * 100
	assert.True(t, CheckBigEqualToInt(250075, odEvent.ExecutedFee)) // 5001500 * 0.05

	log = localStorage.logs[9]
	txEvent := TransactionEvent{}
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, 202, fromOrderIdBytesToInt(txEvent.TakerId))
	assert.Equal(t, 104, fromOrderIdBytesToInt(txEvent.MakerId))
	assert.True(t, CheckBigEqualToInt(4500, txEvent.Quantity))
	assert.True(t, priceEqual(txEvent.Price, "100.04"))
	assert.True(t, CheckBigEqualToInt(45018000, txEvent.Amount))// 4500*100.04 * 100
	assert.True(t, CheckBigEqualToInt(2701080, txEvent.TakerFee)) // 45018000 * 0.06
	assert.True(t, CheckBigEqualToInt(2250900, txEvent.MakerFee)) // 45018000 * 0.05

	log = localStorage.logs[10]
	txEvent = TransactionEvent{}
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, 202, fromOrderIdBytesToInt(txEvent.TakerId))
	assert.Equal(t, 102, fromOrderIdBytesToInt(txEvent.MakerId))
	assert.True(t, CheckBigEqualToInt(500, txEvent.Quantity))
	assert.True(t, priceEqual(txEvent.Price, "100.03"))
	assert.True(t, CheckBigEqualToInt(5001500, txEvent.Amount))
	assert.True(t, CheckBigEqualToInt(300090, txEvent.TakerFee))
	assert.True(t, CheckBigEqualToInt(250075, txEvent.MakerFee))

	buy6 := newOrderInfo(106, ETH, VITE, false, Limited, "100.3", 10100, time.Now().UnixNano()/1000)
	mc.MatchOrder(buy6)
	bookIdForBuy := getBookIdToMakeForTaker(buy6)
	bookIdForSell := getBookIdToTake(buy6)
	assert.Equal(t,106, fromOrderIdToInt(mc.books[bookIdForBuy].header))
	assert.Equal(t,0, fromOrderIdToInt(mc.books[bookIdForSell].header))
	assert.Equal(t,int32(5), mc.books[bookIdForBuy].length)
	assert.Equal(t,int32(0), mc.books[bookIdForSell].length)

	assert.Equal(t, 14, len(localStorage.logs))

	// localStorage.logs[11] orderEvent(106) newTaker 10100
	// localStorage.logs[12] orderEvent(201) makerFilled 10000
	// localStorage.logs[13] txEvent taker -> maker[106, 201]
	log = localStorage.logs[11]
	newOrderEvent = NewOrderEvent{}
	newOrderEvent = newOrderEvent.FromBytes(log.Data).(NewOrderEvent)
	assert.Equal(t, 106, fromOrderIdBytesToInt(newOrderEvent.Order.Id))
	assert.Equal(t, PartialExecuted, int(newOrderEvent.Order.Status))
	assert.True(t, CheckBigEqualToInt(10000, newOrderEvent.Order.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(100100000, newOrderEvent.Order.ExecutedAmount)) // 10000 * 100.1 * 100
	assert.True(t, CheckBigEqualToInt(6006000, newOrderEvent.Order.ExecutedFee)) // 100100000 * 0.06

	log = localStorage.logs[12]
	odEvent = OrderUpdateEvent{}
	odEvent = odEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 201, fromOrderIdBytesToInt(odEvent.Id))
	assert.Equal(t, FullyExecuted, int(odEvent.Status))
	assert.True(t, CheckBigEqualToInt(10000, odEvent.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(100100000, odEvent.ExecutedAmount))
	assert.True(t, CheckBigEqualToInt(5005000, odEvent.ExecutedFee)) // 100100000 * 0.05

	log = localStorage.logs[13]
	txEvent = TransactionEvent{}
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, 106, fromOrderIdBytesToInt(txEvent.TakerId))
	assert.Equal(t, 201, fromOrderIdBytesToInt(txEvent.MakerId))
	assert.True(t, CheckBigEqualToInt(10000, txEvent.Quantity))
	assert.True(t, priceEqual(txEvent.Price, "100.1"))
	assert.True(t, CheckBigEqualToInt(100100000, txEvent.Amount))
	assert.True(t, CheckBigEqualToInt(6006000, txEvent.TakerFee))
	assert.True(t, CheckBigEqualToInt(5005000, txEvent.MakerFee))

	buy7 := newOrderInfo(105, ETH, VITE, false, Limited, "10.01", 20, time.Now().UnixNano()/1000)
	err = mc.MatchOrder(buy7)
	assert.Equal(t, err.Error(), "order id already exists")
}

func TestFeeCalculation(t *testing.T) {
	SetFeeRate("0.07", "0.05") // takerFee, makerFee
	buyTakerOrder := newOrderInfo(601, ETH, VITE, false, Limited, "0.001234211", 1000000, time.Now().UnixNano()/1000)
	assert.True(t, CheckBigEqualToInt(8639, buyTakerOrder.Order.LockedBuyFee)) // 123421 * 0.07

	buyTakerOrder1 := newOrderInfo(602, ETH, VITE, false, Limited, "0.001234231", 1000000, time.Now().UnixNano()/1000)
	assert.True(t, CheckBigEqualToInt(8640, buyTakerOrder1.Order.LockedBuyFee)) //123423 * 0.07
	buyTakerOrder1.Order.ExecutedFee = new(big.Int).SetInt64(8634).Bytes()// 8640 - 8634 = 6
	feeBytes, executedFee := calculateFeeAndExecutedFee(buyTakerOrder1.Order,	new(big.Int).SetInt64(100).Bytes(), "0.07")
	assert.True(t, CheckBigEqualToInt(6, feeBytes))
	assert.True(t, CheckBigEqualToInt(8640, executedFee))
}

func TestDustCheck(t *testing.T) {
	order := &proto.Order{}
	order.Quantity = big.NewInt(2000).Bytes()
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.Price = "0.1"
	tokenInfo := &proto.OrderTokenInfo{}
	tokenInfo.TradeTokenDecimals = 10
	tokenInfo.QuoteTokenDecimals = 8
	assert.False(t, isDust(order, big.NewInt(1000).Bytes(), tokenInfo))
	assert.True(t, isDust(order, big.NewInt(1001).Bytes(), tokenInfo))

	order1 := &proto.Order{}
	order1.Quantity = big.NewInt(1000).Bytes()
	order1.ExecutedQuantity = big.NewInt(0).Bytes()
	order1.Price = "0.001"
	tokenInfo.TradeTokenDecimals = 8
	tokenInfo.QuoteTokenDecimals = 10
	assert.False(t, isDust(order1, big.NewInt(990).Bytes(), tokenInfo))
	assert.True(t, isDust(order1, big.NewInt(991).Bytes(), tokenInfo))
}

func TestDustWithOrder(t *testing.T) {
	localStorage := NewMapStorage()
	st := BaseStorage(&localStorage)
	mc := NewMatcher(getAddress(), &st)
	SetFeeRate("0.06", "0.05") // takerFee, makerFee
	// buy quantity = origin * 100,000,000
	buy1 := newOrderInfo(301, ETH, VITE,false, Limited, "0.00012345", 10000, time.Now().UnixNano()/1000) //amount 123.45
	mc.MatchOrder(buy1)

	bookNameToMakeForBuy := getBookIdToMakeForTaker(buy1)
	buy1New, err := mc.GetOrderByIdAndBookId(bookNameToMakeForBuy, orderIdFromInt(301).bytes())
	//fmt.Printf("err %v\n", err.Error())
	assert.True(t, err == nil)
	assert.True(t, CheckBigEqualToInt(7, buy1New.Order.LockedBuyFee))

	// sell
	sell1 := newOrderInfo(401, ETH, VITE,true, Limited, "0.00012342", 10002, time.Now().UnixNano()/1000) // amount 123.44
	mc.MatchOrder(sell1)
	// sell order.Quantity 10002 order.ExecutedQuantity 10000
	// roundAmount((10002-10000) * 0.00012345 * 100) = 0

	bookNameToMakeForSell := getBookIdToMakeForTaker(sell1)
	assert.Equal(t,int32(0), mc.books[bookNameToMakeForBuy].length)
	assert.Equal(t,int32(0), mc.books[bookNameToMakeForSell].length)
	assert.Equal(t,0, fromOrderIdToInt(mc.books[bookNameToMakeForBuy].header))
	assert.Equal(t,0, fromOrderIdToInt(mc.books[bookNameToMakeForSell].header))

	// localStorage.logs[0] orderEvent(301) newMaker
	// localStorage.logs[1] orderEvent(401) newTaker
	// localStorage.logs[2] orderEvent(301) makerFilled
	// localStorage.logs[3] txEvent taker -> maker[106, 201]
	assert.Equal(t, 4, len(localStorage.logs))
	log := localStorage.logs[1]
	newOrderEvent := NewOrderEvent{} //taker
	newOrderEvent = newOrderEvent.FromBytes(log.Data).(NewOrderEvent)
	assert.Equal(t, 401, fromOrderIdBytesToInt(newOrderEvent.Order.Id))
	assert.Equal(t, FullyExecuted, int(newOrderEvent.Order.Status))
	assert.True(t, CheckBigEqualToInt(10000, newOrderEvent.Order.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(123, newOrderEvent.Order.ExecutedAmount))
	assert.True(t, CheckBigEqualToInt(7, newOrderEvent.Order.ExecutedFee))
	assert.Equal(t, ETH.tokenId.Bytes(), newOrderEvent.Order.RefundToken)
	assert.True(t, CheckBigEqualToInt(2, newOrderEvent.Order.RefundQuantity))

	log = localStorage.logs[2]
	orderEvent := OrderUpdateEvent{} // maker
	orderEvent = orderEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 301, fromOrderIdBytesToInt(orderEvent.Id))
	assert.Equal(t, FullyExecuted, int(orderEvent.Status))
	assert.True(t, CheckBigEqualToInt(10000, orderEvent.ExecutedQuantity))
	assert.True(t, CheckBigEqualToInt(123, orderEvent.ExecutedAmount))
	assert.True(t, CheckBigEqualToInt(6, orderEvent.ExecutedFee))
	assert.Equal(t, VITE.tokenId.Bytes(), orderEvent.RefundToken)
	assert.True(t, CheckBigEqualToInt(1, orderEvent.RefundQuantity)) // LockedBuyFee - ExecutedFee



	log = localStorage.logs[3]
	txEvent := TransactionEvent{} // maker
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, true, txEvent.TakerSide)
	assert.Equal(t, 401, fromOrderIdBytesToInt(txEvent.TakerId))
	assert.Equal(t, 301, fromOrderIdBytesToInt(txEvent.MakerId))
	assert.True(t, priceEqual("0.00012345", txEvent.Price))
	assert.True(t, CheckBigEqualToInt(10000, txEvent.Quantity))
	assert.True(t, CheckBigEqualToInt(123, txEvent.Amount))
	assert.True(t, CheckBigEqualToInt(7, txEvent.TakerFee))
	assert.True(t, CheckBigEqualToInt(6, txEvent.MakerFee))
}

func TestMarket(t *testing.T) {

}

func newOrderInfo(id int, tradeToken tokenInfo, quoteToken tokenInfo, side bool, orderType int32, price string, quantity uint64, ts int64) TakerOrder {
	tokenInfo := &proto.OrderTokenInfo{}
	tokenInfo.TradeToken = tradeToken.tokenId.Bytes()
	tokenInfo.QuoteToken = quoteToken.tokenId.Bytes()
	tokenInfo.TradeTokenDecimals = tradeToken.decimals
	tokenInfo.QuoteTokenDecimals = quoteToken.decimals
	order := &proto.Order{}
	order.Id = orderIdBytesFromInt(id)
	order.Side = side // buy
	order.Type = orderType
	order.Price = price
	order.Quantity = new(big.Int).SetUint64(quantity).Bytes()
	order.Status = Pending
	order.Amount = CalculateRawAmount(order.Quantity, order.Price, tokenInfo.TradeTokenDecimals, tokenInfo.QuoteTokenDecimals)
	if order.Type == Limited && !order.Side {//buy
		//fmt.Printf("newOrderInfo set LockedBuyFee id %v, order.Type %v, order.Side %v, order.Amount %v\n", id, order.Type, order.Side, order.Amount)
		order.LockedBuyFee = CalculateRawFee(order.Amount, MaxFeeRate())
	}
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
	orderInfo := proto.OrderInfo{Order:order, OrderTokenInfo:tokenInfo}
	return TakerOrder{orderInfo}
}

func CheckBigEqualToInt(expected int, value []byte) bool {
	return new(big.Int).SetUint64(uint64(expected)).Cmp(new(big.Int).SetBytes(value)) == 0
}

func fromOrderIdToInt(orderId nodeKeyType) int {
	return fromOrderIdBytesToInt(orderId.(OrderId).bytes())
}

func fromOrderIdBytesToInt(orderIdBytes []byte) int {
	res := int(binary.BigEndian.Uint32(orderIdBytes[16:20]))
	return res
}