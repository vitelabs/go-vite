package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
)

const maxTakedOrders = 1000

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
		emitOrderRes(taker)
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
	matchedMakers := make([]Order, 0, 20)
	if maker, nextOrderId, err := getMakerById(makerBook, makerBook.header); err != nil {
		return false, err
	} else {
		if err := mc.recursiveTakeOrder(&taker, maker, makerBook, matchedMakers, nextOrderId); err != nil {
			return false, err
		} else {
			mc.handleTaker(taker)
			mc.handleMatchedMakers(matchedMakers, makerBook)
			if len(matchedMakers) > 0 {
				return true, nil
			} else {
				return false, nil
			}
		}
	}
}

func (mc *matcher) recursiveTakeOrder(taker *Order, maker Order, makerBook *skiplist, matchedMakers []Order, orderId nodeKeyType)  error {
	matched, executedPrice := matchPrice(*taker, maker)
	if !matched {
		return nil
	} else {
		
		matchedMakers = append(matchedMakers, maker)
	}
	machedSize := len(matchedMakers)
	if taker.Status == partialExecuted && (machedSize >= maxTakedOrders || machedSize == makerBook.length) {
		taker.Status = partialExecutedCancelledByMarket
	}
	if taker.Status != partialExecuted {
		return nil
	}
	var	err error
	if maker, orderId, err = getMakerById(makerBook, orderId); err != nil {

	}
	return mc.recursiveTakeOrder(taker, maker, makerBook, matchedMakers, orderId)
}

func getMakerById(makerBook *skiplist, orderId nodeKeyType) (od Order, nextId orderKey, err error) {
	if pl, fwk, _, err := makerBook.getByKey(orderId); err != nil {
		return Order{}, (*makerBook.protocol).getNilKey().(orderKey), err
	} else {
		maker := (*pl).(Order)
		return maker, fwk.(orderKey), nil
	}
}

func (mc *matcher) handleMatchedMakers(matchedMakers []Order, makerBook *skiplist) {
	for _, maker := range matchedMakers {
		pl := nodePayload(maker)
		makerBook.updatePayload(newOrderKey(maker.Id), &pl)
		emitOrderRes(maker)
	}
	size := len(matchedMakers)
	if matchedMakers[size-1].Status == fullyExecuted {
		makerBook.truncateHeadTo(newOrderKey(matchedMakers[size-1].Id), size)
	} else {
		makerBook.truncateHeadTo(newOrderKey(matchedMakers[size-2].Id), size-1)
	}
}

func (mc *matcher) handleTaker(order Order) {
	if order.Status == partialExecuted {
		mc.saveAsMaker(order)
	}
	emitOrderRes(order)
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

func emitOrderRes(orderRes Order) {

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
