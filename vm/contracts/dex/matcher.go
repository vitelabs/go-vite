package dex

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"time"
)

const maxTxsCountPerTaker = 1000
const timeoutMillisecond = 7 * 24 * 3600 * 1000

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
	if taker, ok = validateAndRenderOrder(order); !ok {
		mc.emitOrderRes(taker)
		return
	}
	bookNameToTake := getBookNameToTake(taker)
	if bookToTake, ok = mc.books[bookNameToTake]; !ok {
		bookToTake = newSkiplist(bookNameToTake, mc.contractAddress, mc.storage, mc.protocol)
	}
	mc.books[bookNameToTake] = bookToTake
	if err := mc.doMatchTaker(taker, bookToTake); err != nil {
		fmt.Printf("Failed to match taker for %s", err.Error())
	}
}

func (mc *matcher) doMatchTaker(taker Order, makerBook *skiplist) error {
	if makerBook.length == 0 {
		mc.handleTakerRes(taker)
		return nil
	}
	modifiedMakers := make([]Order, 0, 20)
	txs := make([]proto.Transaction, 0, 20)
	if maker, nextOrderId, err := getMakerById(makerBook, makerBook.header); err != nil {
		return err
	} else {
		if err := mc.recursiveTakeOrder(&taker, maker, makerBook, &modifiedMakers, &txs, nextOrderId); err != nil {
			return err
		} else {
			mc.handleTakerRes(taker)
			mc.handleModifiedMakers(modifiedMakers, makerBook)
			mc.emitTxs(txs)
			return nil
		}
	}
}
//TODO add assertion for order calculation correctness
func (mc *matcher) recursiveTakeOrder(taker *Order, maker Order, makerBook *skiplist, modifiedMakers *[]Order, txs *[]proto.Transaction, nextOrderId nodeKeyType) error {
	if filterTimeout(&maker) {
		calculateRefund(&maker)
		*modifiedMakers = append(*modifiedMakers, maker)
	} else {
		matched, _ := matchPrice(*taker, maker)
		fmt.Printf("recursiveTakeOrder matched for taker.id %d is %t\n", taker.Id, matched)
		if matched {
			tx := calculateOrderAndTx(taker, &maker)
			calculateRefund(taker)
			calculateRefund(&maker)
			*modifiedMakers = append(*modifiedMakers, maker)
			*txs = append(*txs, tx)
		}
	}
	if taker.Status == partialExecuted && len(*txs) >= maxTxsCountPerTaker {
		taker.Status = cancelled
		taker.CancelReason = partialExecutedCancelledByMarket
	}
	if taker.Status == fullyExecuted || taker.Status == cancelled {
		return nil
	}
	// current maker is the last item in the book
	if makerBook.tail.(orderKey).value == maker.Id {
		return nil
	}
	var err error
	if maker, nextOrderId, err = getMakerById(makerBook, nextOrderId); err != nil {
		return errors.New("Failed get order by nextOrderId")
	} else {
		return mc.recursiveTakeOrder(taker, maker, makerBook, modifiedMakers, txs, nextOrderId)
	}
}

func (mc *matcher) handleModifiedMakers(makers []Order, makerBook *skiplist) {
	for _, maker := range makers {
		pl := nodePayload(maker)
		if err := makerBook.updatePayload(newOrderKey(maker.Id), &pl); err != nil {
			fmt.Printf("failed update maker storage for err : %s\n", err.Error())
		}
		mc.emitOrderRes(maker)
	}
	size := len(makers)
	if size > 0 {
		if makers[size-1].Status == fullyExecuted || makers[size-1].Status == cancelled {
			makerBook.truncateHeadTo(newOrderKey(makers[size-1].Id), size)
		} else if size >= 2 {
			makerBook.truncateHeadTo(newOrderKey(makers[size-2].Id), size-1)
		}
	}

}

func (mc *matcher) handleTakerRes(order Order) {
	if order.Status == partialExecuted || order.Status == pending {
		mc.saveTakerAsMaker(order)
	}
	mc.emitOrderRes(order)
}

func (mc *matcher) saveTakerAsMaker(maker Order) {
	var (
		bookToMake *skiplist
		ok         bool
	)
	bookNameToMake := getBookNameToMake(maker)
	if bookToMake, ok = mc.books[bookNameToMake]; !ok {
		bookToMake = newSkiplist(bookNameToMake, mc.contractAddress, mc.storage, mc.protocol)
		mc.books[bookNameToMake] = bookToMake
	}
	pl := nodePayload(maker)
	bookToMake.insert(newOrderKey(maker.Id), &pl)
}

func (mc *matcher) emitOrderRes(orderRes Order) {
	event := OrderUpdateEvent{orderRes.Order}
	(*mc.storage).AddLog(newLog(event))
	fmt.Printf("order matched res %s : \n", orderRes.String())
}

func (mc *matcher) emitTxs(txs []proto.Transaction) {
	fmt.Printf("matched txs >>>>>>>>> %d\n", len(txs))
	for _, tx := range txs {
		txEvent := TransactionEvent{tx}
		(*mc.storage).AddLog(newLog(txEvent))
		fmt.Printf("matched tx is : %s\n", tx.String())
	}
}

func newLog(event OrderEvent) *ledger.VmLog {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.getTopicId())
	log.Data = event.toDataBytes()
	return log
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

func filterTimeout(maker *Order) bool {
	if time.Now().Unix()*1000 > maker.Timestamp+timeoutMillisecond {
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
	if order.Status == fullyExecuted || order.Status == cancelled {
		switch order.Side {
		case false: //buy
			order.RefundAsset = order.QuoteAsset
			order.RefundQuantity = order.Amount - order.ExecutedAmount
		case true:
			order.RefundAsset = order.TradeAsset
			order.RefundQuantity = order.Quantity - order.ExecutedQuantity
		}
	}
}

func calculateOrderAndTx(taker *Order, maker *Order) (tx proto.Transaction) {
	tx = proto.Transaction{}
	tx.Id = getTxId(taker.Id, maker.Id)
	tx.TakerSide = taker.Side
	tx.TakerId = taker.Id
	tx.MakerId = maker.Id
	tx.Price = maker.Price
	executeQuantity := minUint64(taker.Quantity-taker.ExecutedQuantity, maker.Quantity-maker.ExecutedQuantity)
	takerAmount := calculateOrderAmount(taker, executeQuantity, maker.Price)
	makerAmount := calculateOrderAmount(maker, executeQuantity, maker.Price)
	executeAmount := minUint64(takerAmount, makerAmount)
	executeQuantity = minUint64(executeQuantity, calculateQuantity(executeAmount, maker.Price))
	fmt.Printf("calculateOrderAndTx >>> taker.ExecutedQuantity : %d, maker.ExecutedQuantity : %d, maker.Price : %f, executeQuantity : %d, executeAmount : %d \n",
		taker.ExecutedQuantity,
		maker.ExecutedQuantity,
		maker.Price,
		executeQuantity,
		executeAmount)
	updateOrder(taker, executeQuantity, executeAmount)
	updateOrder(maker, executeQuantity, executeAmount)
	tx.Quantity = executeQuantity
	tx.Amount = executeAmount
	tx.Timestamp = time.Now().UnixNano() / 1000
	return tx
}

func calculateOrderAmount(order *Order, quantity uint64, price float64) uint64 {
	amount := calculateAmount(quantity, price)
	if !order.Side && order.Amount < order.ExecutedAmount+amount {// side is buy
		amount = order.Amount - order.ExecutedAmount
	}
	return amount
}

func updateOrder(order *Order, quantity uint64, amount uint64) uint64 {
	order.ExecutedAmount += amount
	if order.Quantity-order.ExecutedQuantity == quantity || isDust(order, quantity) {
		order.Status = fullyExecuted
	} else {
		order.Status = partialExecuted
	}
	order.ExecutedQuantity += quantity
	return amount
}
// leave quantity is too small for calculate precision
func isDust(order *Order, quantity uint64) bool {
	return calculateAmount(order.Quantity-order.ExecutedQuantity - quantity, order.Price) < 1
}

func getMakerById(makerBook *skiplist, orderId nodeKeyType) (od Order, nextId orderKey, err error) {
	if pl, fwk, _, err := makerBook.getByKey(orderId); err != nil {
		return Order{}, (*makerBook.protocol).getNilKey().(orderKey), err
	} else {
		maker := (*pl).(Order)
		return maker, fwk.(orderKey), nil
	}
}

func validateAndRenderOrder(order Order) (orderRes Order, isValid bool) {
	if order.Id == 0 || !validPrice(order.Price) {
		order.Status = cancelled
		order.CancelReason = cancelledByMarket
		return order, false
	} else {
		order.Status = pending
		if order.Amount == 0 {
			order.Amount = calculateAmount(order.Quantity, order.Price)
		}
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

func calculateAmount(quantity uint64, price float64) uint64 {
	qtF := big.NewFloat(0).SetUint64(quantity)
	prF := big.NewFloat(0).SetFloat64(price)
	amountF := prF.Mul(prF, qtF)
	amount, _ := amountF.Add(amountF, big.NewFloat(0.5)).Uint64()
	return amount
}

func calculateQuantity(amount uint64, price float64) uint64 {
	amtF := big.NewFloat(0).SetUint64(amount)
	prF := big.NewFloat(0).SetFloat64(price)
	qtyF := amtF.Quo(amtF, prF)
	qty, _ := qtyF.Add(qtyF, big.NewFloat(0.5)).Uint64()
	return qty
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

func getTxId(takerId uint64, makerId uint64) uint64 {
	data := crypto.Hash(8, []byte(fmt.Sprintf("%d-%d", takerId, makerId)))
	var num uint64
	binary.Read(bytes.NewBuffer(data[:]), binary.LittleEndian, &num)
	return num
}
