package dex

import (
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
	"time"
)

var (
	VITE = types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}
	ETH = types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}
	NANO = types.TokenTypeId{'N', 'A', 'N', 'O', ' ', 'T', 'O', 'K', 'E', 'N'}
)

func TestMatcher(t *testing.T) {
	localStorage := NewMapStorage()
	st := baseStorage(&localStorage)
	var po nodePayloadProtocol = &OrderNodeProtocol{}
	mc := newMatcher(getAddress(), &st, &po)
	// buy
	buy5 := newOrderInfo(105, ETH, VITE, false, Limited, 100.01, 20, time.Now().UnixNano()/1000)
	buy3 := newOrderInfo(103, ETH, VITE, false, Limited, 100.02, 70, time.Now().UnixNano()/1000)
	buy4 := newOrderInfo(104, ETH, VITE, false, Limited, 100.04, 45, time.Now().UnixNano()/1000)
	buy2 := newOrderInfo(102, ETH, VITE, false, Limited, 100.03, 30, time.Now().UnixNano()/1000)
	buy1 := newOrderInfo(101, ETH, VITE, false, Limited, 100.02, 10, time.Now().UnixNano()/1000)
	mc.matchOrder(buy1)
	mc.matchOrder(buy2)
	mc.matchOrder(buy3)
	mc.matchOrder(buy4)
	mc.matchOrder(buy5)
	assert.Equal(t, 5, len(localStorage.logs))

	bookNameToMakeForBuy := getBookNameToMake(buy5)
	assert.Equal(t,104, int(mc.books[bookNameToMakeForBuy].header.(orderKey).value))

	sell1 := newOrderInfo(201, ETH, VITE, true, Limited, 100.1, 100, time.Now().UnixNano()/1000)
	sell2 := newOrderInfo(202, ETH, VITE, true, Limited, 100.02, 50, time.Now().UnixNano()/1000)
	mc.matchOrder(sell1)
	assert.Equal(t, 6, len(localStorage.logs))
	mc.matchOrder(sell2)

	bookNameToMakeForSell := getBookNameToMake(sell1)
	assert.Equal(t,201, int(mc.books[bookNameToMakeForSell].header.(orderKey).value))
	assert.Equal(t,102, int(mc.books[bookNameToMakeForBuy].header.(orderKey).value))
	assert.Equal(t,4, mc.books[bookNameToMakeForBuy].length)
	assert.Equal(t,1, mc.books[bookNameToMakeForSell].length)

	assert.Equal(t, 11, len(localStorage.logs))
	pl, _, _, _  := mc.books[bookNameToMakeForSell].getByKey(mc.books[bookNameToMakeForBuy].header)
	od, _ := (*pl).(Order)
	assert.Equal(t, 5, int(od.ExecutedQuantity))

	log := localStorage.logs[9]
	txEvent := TransactionEvent{}
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, 202, int(txEvent.TakerId))
	assert.Equal(t, 104, int(txEvent.MakerId))
	assert.Equal(t, 45, int(txEvent.Quantity))
	assert.True(t, priceEqual(txEvent.Price, float64(100.04)))
	assert.Equal(t, uint64(4502), txEvent.Amount)

	log = localStorage.logs[10]
	txEvent = TransactionEvent{}
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, 202, int(txEvent.TakerId))
	assert.Equal(t, 102, int(txEvent.MakerId))
	assert.Equal(t, 5, int(txEvent.Quantity))
	assert.True(t, priceEqual(txEvent.Price, 100.03))
	assert.Equal(t, uint64(500), txEvent.Amount)

	buy6 := newOrderInfo(106, ETH, VITE, false, Limited, 100.3, 101, time.Now().UnixNano()/1000)
	mc.matchOrder(buy6)
	bookNameForBuy := getBookNameToMake(buy6)
	bookNameForSell := getBookNameToTake(buy6)
	assert.Equal(t,106, int(mc.books[bookNameForBuy].header.(orderKey).value))
	assert.Equal(t,0, int(mc.books[bookNameForSell].header.(orderKey).value))
	assert.Equal(t,5, mc.books[bookNameForBuy].length)
	assert.Equal(t,0, mc.books[bookNameForSell].length)

	assert.Equal(t, 14, len(localStorage.logs))

	log = localStorage.logs[11]
	odEvent := OrderUpdateEvent{}
	odEvent = odEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 106, int(odEvent.Id))
	assert.Equal(t, PartialExecuted, int(odEvent.Status))
	assert.Equal(t, 100, int(odEvent.ExecutedQuantity))
	assert.Equal(t, 10010, int(odEvent.ExecutedAmount))

	log = localStorage.logs[12]
	odEvent = OrderUpdateEvent{}
	odEvent = odEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 201, int(odEvent.Id))
	assert.Equal(t, FullyExecuted, int(odEvent.Status))
	assert.Equal(t, 100, int(odEvent.ExecutedQuantity))
	assert.Equal(t, 10010, int(odEvent.ExecutedAmount))

	log = localStorage.logs[13]
	txEvent = TransactionEvent{}
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, 106, int(txEvent.TakerId))
	assert.Equal(t, 201, int(txEvent.MakerId))
	assert.Equal(t, 100, int(txEvent.Quantity))
	assert.True(t, priceEqual(txEvent.Price, 100.1))
	assert.Equal(t, uint64(10010), txEvent.Amount)
}

func TestDust(t *testing.T) {
	localStorage := NewMapStorage()
	st := baseStorage(&localStorage)
	var po nodePayloadProtocol = &OrderNodeProtocol{}
	mc := newMatcher(getAddress(), &st, &po)
	// buy quantity = origin * 100,000,000
	buy1 := newOrderInfo(301, VITE, ETH, false, Limited, float64(0.0012345), 100000000, time.Now().UnixNano()/1000)
	mc.matchOrder(buy1)
	// sell
	sell1 := newOrderInfo(401, VITE, ETH,true, Limited, float64(0.0012342), 100000200, time.Now().UnixNano()/1000)
	mc.matchOrder(sell1)

	bookNameToMakeForBuy := getBookNameToMake(buy1)
	bookNameToMakeForSell := getBookNameToMake(sell1)
	assert.Equal(t,0, mc.books[bookNameToMakeForBuy].length)
	assert.Equal(t,0, mc.books[bookNameToMakeForSell].length)
	assert.Equal(t,0, int(mc.books[bookNameToMakeForBuy].header.(orderKey).value))
	assert.Equal(t,0, int(mc.books[bookNameToMakeForSell].header.(orderKey).value))

	assert.Equal(t, 4, len(localStorage.logs))
	log := localStorage.logs[1]
	orderEvent := OrderUpdateEvent{} //taker
	orderEvent = orderEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 401, int(orderEvent.Id))
	assert.Equal(t, FullyExecuted, int(orderEvent.Status))
	assert.Equal(t, uint64(100000000), orderEvent.ExecutedQuantity)
	assert.Equal(t, uint64(123450), orderEvent.ExecutedAmount)
	assert.Equal(t, VITE.Bytes(), orderEvent.RefundAsset)
	assert.Equal(t, uint64(200), orderEvent.RefundQuantity)

	log = localStorage.logs[2]
	orderEvent = OrderUpdateEvent{} // maker
	orderEvent = orderEvent.fromBytes(log.Data).(OrderUpdateEvent)
	assert.Equal(t, 301, int(orderEvent.Id))
	assert.Equal(t, FullyExecuted, int(orderEvent.Status))
	assert.Equal(t, uint64(100000000), orderEvent.ExecutedQuantity)
	assert.Equal(t, uint64(123450), orderEvent.ExecutedAmount)
	assert.Equal(t, ETH.Bytes(), orderEvent.RefundAsset)
	assert.Equal(t, uint64(0), orderEvent.RefundQuantity)

	log = localStorage.logs[3]
	txEvent := TransactionEvent{} // maker
	txEvent = txEvent.fromBytes(log.Data).(TransactionEvent)
	assert.Equal(t, true, txEvent.TakerSide)
	assert.Equal(t, 401, int(txEvent.TakerId))
	assert.Equal(t, 301, int(txEvent.MakerId))
	assert.Equal(t, float64(0.0012345), txEvent.Price)
	assert.Equal(t, uint64(100000000), txEvent.Quantity)
	assert.Equal(t, uint64(123450), txEvent.Amount)
}

func TestMarket(t *testing.T) {

}


func newOrderInfo(id uint64, tradeAsset types.TokenTypeId, quoteAsset types.TokenTypeId, side bool, orderType uint32, price float64, quantity uint64, ts int64) Order {
	order := Order{}
	order.Id = uint64(id)
	order.TradeAsset = tradeAsset.Bytes()
	order.QuoteAsset = quoteAsset.Bytes()
	order.Side = side // buy
	order.Type = orderType
	order.Price = price
	order.Quantity = quantity
	order.Timestamp = int64(ts)
	return order
}