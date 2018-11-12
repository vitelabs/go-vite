package dex

import (
	"fmt"
	"github.com/Loopring/relay/log"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"time"
)

const maxTxsCountPerTaker = 1000
const timeoutMillisecond = 7*24*3600*1000

type matcher struct {
	contractAddress *types.Address
	storage         *baseStorage
	protocol        *nodePayloadProtocol
	books           map[string]*skiplist
}

func newMatcher(contractAddress *types.Address, storage *baseStorage, protocol *nodePayloadProtocol) *matcher {
	mc := &matcher{}
	mc.contractAddress = contractAddress
	mc.storage = storage
	mc.protocol = protocol
	mc.books = make(map[string]*skiplist)
	return mc
}

func (mc *matcher) matchOrder(order Order) {
	var (
		bookToTake *skiplist
		taker      Order
		ok         bool
	)
	if taker, ok = validAndRenderOrder(order); !ok {
		mc.emitOrderRes(taker)
		return
	}
	bookNameToTake := getBookNameToTake(taker)
	if bookToTake, ok = mc.books[bookNameToTake]; !ok {
		bookToTake = newSkiplist(bookNameToTake, mc.contractAddress, mc.storage, mc.protocol)
	}
	mc.doMatchTaker(taker, bookToTake)
}

func (mc *matcher) doMatchTaker(taker Order, makerBook *skiplist) (matched bool, err error) {
	if makerBook.length == 0 {
		return false, nil
	}
	modifiedMakers := make([]Order, 0, 20)
	txs := make([]proto.Transaction, 0, 20)
	if maker, nextOrderId, err := getMakerById(makerBook, makerBook.header); err != nil {
		return false, err
	} else {
		if err := mc.recursiveTakeOrder(&taker, maker, makerBook, modifiedMakers, txs, nextOrderId); err != nil {
			return false, err
		} else {
			mc.handleTaker(taker)
			mc.handleModifieddMakers(modifiedMakers, makerBook)
			mc.emitTxs(txs)
			if len(txs) > 0 {
				return true, nil
			} else {
				return false, nil
			}
		}
	}
}

func (mc *matcher) recursiveTakeOrder(taker *Order, maker Order, makerBook *skiplist, modifiedMakers []Order, txs []proto.Transaction, orderId nodeKeyType)  error {
	if filterTimeout(&maker) {
		calculateRefund(&maker)
		modifiedMakers = append(modifiedMakers, maker)
	} else {
		matched, _ := matchPrice(*taker, maker)
		if matched {
			tx := calculateOrderAndTx(taker, &maker)
			calculateRefund(taker)
			calculateRefund(&maker)
			modifiedMakers = append(modifiedMakers, maker)
			txs = append(txs, tx)
		}
	}
	machedSize := len(txs)
	if taker.Status == partialExecuted && machedSize >= maxTxsCountPerTaker {
		taker.Status = partialExecutedCancelledByMarket
	}
	if taker.Status == fullyExecuted || taker.Status == cancelled {
		return nil
	}
	if makerBook.tail.(orderKey).value == maker.Id {
		return nil
	}
	var err error
	if maker, orderId, err = getMakerById(makerBook, orderId); err != nil {
		log.Fatal("Failed get order by orderId")
		return nil
	} else {
		return mc.recursiveTakeOrder(taker, maker, makerBook, modifiedMakers, txs, orderId)
	}
}

func (mc *matcher) handleModifieddMakers(matchedMakers []Order, makerBook *skiplist) {
	for _, maker := range matchedMakers {
		pl := nodePayload(maker)
		makerBook.updatePayload(newOrderKey(maker.Id), &pl)
		mc.emitOrderRes(maker)
	}
	size := len(matchedMakers)
	if matchedMakers[size-1].Status == fullyExecuted {
		makerBook.truncateHeadTo(newOrderKey(matchedMakers[size-1].Id), size)
	} else {
		makerBook.truncateHeadTo(newOrderKey(matchedMakers[size-2].Id), size-1)
	}
}

func (mc *matcher) handleTaker(order Order) {
	if order.Status == partialExecuted || order.Status == pending {
		mc.saveAsMaker(order)
	}
	mc.emitOrderRes(order)
}

func matchPrice(taker Order, maker Order) (matched bool, executedPrice float64) {
	if taker.Type == market || priceEqual(taker.Price, maker.Price) {
		return true, maker.Price
	} else {
		matched = false
		switch maker.Side {
		case false: // buy
			matched = maker.Price > taker.Price
		case true: // sell
			matched = maker.Price < taker.Price
		}
		return matched, maker.Price
	}
}

func(mc *matcher) emitOrderRes(orderRes Order) {

}

func(mc *matcher) emitTxs(txs []proto.Transaction) {

}

func filterTimeout(maker *Order) bool {
	if time.Now().Unix()*1000 > maker.Timestamp + timeoutMillisecond {
		switch maker.Status {
		case pending:
			maker.CancelReason = cancelledOnTimeout
		case partialExecuted:
			maker.CancelReason = partialExecutedCancelledOnTimeout
		default:
			maker.CancelReason = unknownCancelledOnTimeout
		}
		maker.Status = cancelled
		return true
	} else {
		return false
	}
}

func calculateRefund(order *Order) {
	switch order.Side {
	case false: //buy
		order.refundAsset = order.QuoteAsset
		order.refundQuantity = order.Amount - order.ExecutedAmount
	case true:
		order.refundAsset = order.TradeAsset
		order.refundQuantity = order.Quantity - order.ExecutedQuantity
	}
}

func calculateOrderAndTx(taker *Order, maker *Order) (tx proto.Transaction) {
	tx = proto.Transaction{}
	matchedQuantity := minUint64(taker.Quantity - taker.ExecutedQuantity, maker.Quantity - maker.ExecutedQuantity)
	takerAmount := calculateOrder(taker, matchedQuantity, maker.Price)
	makerAmount := calculateOrder(maker, matchedQuantity, maker.Price)
	actualAmount := minUint64(takerAmount, makerAmount)
	tx.TakerSide = taker.Side
	tx.TakerId = taker.Id
	tx.MakerId = maker.Id
	tx.Price = maker.Price
	tx.Quantity = matchedQuantity
	tx.Amount = actualAmount
	tx.Timestamp = time.Now().UnixNano()/1000
	return tx
}

func calculateOrder(order *Order, quantity uint64, price float64) uint64 {
	qtF := big.NewFloat(0).SetUint64(quantity)
	prF := big.NewFloat(0).SetFloat64(price)
	amountF := prF.Mul(prF, qtF)
	amount , _ := amountF.Add(amountF, big.NewFloat(0.5)).Uint64()
	if order.Amount < order.ExecutedAmount + amount {
		amount = order.Amount - order.ExecutedAmount
	}
	order.ExecutedAmount += amount
	if order.Quantity - order.ExecutedQuantity == quantity {
		order.Status = fullyExecuted
		order.ExecutedQuantity = order.Quantity
	} else {
		order.Status = partialExecuted
		order.ExecutedQuantity += quantity
	}
	return amount
}

func getMakerById(makerBook *skiplist, orderId nodeKeyType) (od Order, nextId orderKey, err error) {
	if pl, fwk, _, err := makerBook.getByKey(orderId); err != nil {
		return Order{}, (*makerBook.protocol).getNilKey().(orderKey), err
	} else {
		maker := (*pl).(Order)
		return maker, fwk.(orderKey), nil
	}
}

func validAndRenderOrder(order Order) (orderRes Order, isValid bool) {
	if order.Id == 0 || !validPrice(order.Price) {
		order.Status = cancelledByMarket
		return order, false
	} else {
		order.Status = pending
		return order, true
	}
}

func validPrice(price float64) bool {
	if price < 0 || price < 0.00000001 {
		return false
	} else {
		return true
	}
}

func (mc matcher) saveAsMaker(maker Order) {
	var (
		bookToMake *skiplist
		ok         bool
	)
	bookNameToMake := getBookNameToMake(maker)
	if bookToMake, ok = mc.books[bookNameToMake]; !ok {
		bookToMake = newSkiplist(bookNameToMake, mc.contractAddress, mc.storage, mc.protocol)
	}
	pl := nodePayload(maker)
	bookToMake.insert(newOrderKey(maker.Id), &pl)
}

func getBookNameToTake(order Order) string {
	return fmt.Sprintf("%d|%d|%d", order.TradeAsset, order.QuoteAsset, 1-toSideInt(order.Side))
}

func getBookNameToMake(order Order) string {
	return fmt.Sprintf("%d|%d|%d", order.TradeAsset, order.QuoteAsset, toSideInt(order.Side))
}

func toSideInt(side bool) int {
	sideInt := 0 // buy
	if side {
		sideInt = 1 // sell
	}
	return sideInt
}

func minUint64(a uint64, b uint64) uint64 {
	if a > b {
		return b
	} else {
		return a
	}
}