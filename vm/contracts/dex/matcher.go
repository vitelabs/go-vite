package dex

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"strings"
)

const maxTxsCountPerTaker = 1000
const timeoutSecond = 7 * 24 * 3600
const txIdLength = 20

type matcher struct {
	contractAddress *types.Address
	storage         *BaseStorage
	protocol        *nodePayloadProtocol
	books           map[SkipListId]*skiplist
	settleActions   map[types.Address]map[types.TokenTypeId]*proto.SettleAction
}

type OrderTx struct {
	proto.Transaction
	takerAddress    []byte
	makerAddress    []byte
	takerTradeToken []byte
	takerQuoteToken []byte
}

func NewMatcher(contractAddress *types.Address, storage *BaseStorage) *matcher {
	mc := &matcher{}
	mc.contractAddress = contractAddress
	mc.storage = storage
	var po nodePayloadProtocol = &OrderNodeProtocol{}
	mc.protocol = &po
	mc.books = make(map[SkipListId]*skiplist)
	mc.settleActions = make(map[types.Address]map[types.TokenTypeId]*proto.SettleAction)
	return mc
}

func (mc *matcher) MatchOrder(taker Order) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("matchOrder failed with exception %v", r)
		}
	}()
	var bookToTake *skiplist
	if bookToTake, err = mc.getBookById(getBookIdToTake(taker)); err != nil {
		return err
	}
	if err := mc.doMatchTaker(taker, bookToTake); err != nil {
		return err
	}
	return nil
}

func (mc *matcher) GetSettleActions() map[types.Address]map[types.TokenTypeId]*proto.SettleAction {
	return mc.settleActions
}

func (mc *matcher) GetOrderByIdAndBookId(orderIdBytes []byte, makerBookId SkipListId) (*Order, error) {
	var (
		book *skiplist
		err error
	)
	if book, err = mc.getBookById(makerBookId); err != nil {
		return nil, err
	}
	//fmt.Printf("makerBookId %s, orderId %d\n", makerBookId, orderId)
	orderId := OrderId{}
	if err := orderId.setBytes(orderIdBytes); err != nil {
		return nil, err
	}
	if pl, _, _, err := book.getByKey(orderId); err != nil {
		return nil, fmt.Errorf("failed get order by orderId")
	} else {
		od, _ := (*pl).(Order)
		return &od, nil
	}
}

func (mc *matcher) CancelOrderByIdAndBookId(order *Order, makerBookId SkipListId) (err error) {
	var book *skiplist
	if book, err = mc.getBookById(makerBookId); err != nil {
		return err
	}
	switch order.Status {
	case Pending:
		order.CancelReason = cancelledByUser
	case PartialExecuted:
		order.CancelReason = partialExecutedUserCancelled
	}
	order.Status = Cancelled
	mc.handleRefund(order)
	mc.emitOrderRes(*order)
	pl := nodePayload(*order)
	var orderId OrderId
	if orderId, err = NewOrderId(order.Id); err != nil {
		return err
	}
	if err = book.updatePayload(orderId, &pl); err != nil {
		return err
	}
	return book.delete(orderId)
}

func (mc *matcher) getBookById(bookId SkipListId) (*skiplist, error) {
	var (
		book *skiplist
		err error
		ok bool
	)
	if book, ok = mc.books[bookId]; !ok {
		if book, err = newSkiplist(bookId, mc.contractAddress, mc.storage, mc.protocol); err != nil {
			return nil, err
		}
		mc.books[bookId] = book
	}
	return mc.books[bookId], nil
}

func (mc *matcher) doMatchTaker(taker Order, makerBook *skiplist) (err error) {
	if makerBook.length == 0 {
		return mc.handleTakerRes(taker)
	}
	modifiedMakers := make([]Order, 0, 20)
	txs := make([]OrderTx, 0, 20)
	if maker, nextOrderId, err := getMakerById(makerBook, makerBook.header); err != nil {
		return err
	} else {
		if err := mc.recursiveTakeOrder(&taker, maker, makerBook, &modifiedMakers, &txs, nextOrderId); err != nil {
			return err
		} else {
			if err = mc.handleTakerRes(taker); err != nil {
				return err
			}
			if err = mc.handleModifiedMakers(modifiedMakers, makerBook); err != nil {
				return err
			}
			mc.emitTxs(txs)
			return nil
		}
	}
}

//TODO add assertion for order calculation correctness
func (mc *matcher) recursiveTakeOrder(taker *Order, maker Order, makerBook *skiplist, modifiedMakers *[]Order, txs *[]OrderTx, nextOrderId nodeKeyType) error {
	if filterTimeout(taker.Timestamp, &maker) {
		mc.handleRefund(&maker)
		*modifiedMakers = append(*modifiedMakers, maker)
	} else {
		matched, _ := matchPrice(*taker, maker)
		//fmt.Printf("recursiveTakeOrder matched for taker.id %d is %t\n", taker.Id, matched)
		if matched {
			tx := calculateOrderAndTx(taker, &maker)
			mc.handleRefund(taker)
			mc.handleRefund(&maker)
			*modifiedMakers = append(*modifiedMakers, maker)
			*txs = append(*txs, tx)
		}
	}
	if taker.Status == PartialExecuted && len(*txs) >= maxTxsCountPerTaker {
		taker.Status = Cancelled
		taker.CancelReason = partialExecutedCancelledByMarket
	}
	if taker.Status == FullyExecuted || taker.Status == Cancelled {
		return nil
	}
	// current maker is the last item in the book
	makerId, _ := NewOrderId(maker.Id)
	if makerBook.tail.equals(makerId) {
		return nil
	}
	var err error
	if maker, nextOrderId, err = getMakerById(makerBook, nextOrderId); err != nil {
		return errors.New("Failed get order by nextOrderId")
	} else {
		return mc.recursiveTakeOrder(taker, maker, makerBook, modifiedMakers, txs, nextOrderId)
	}
}

func (mc *matcher) handleModifiedMakers(makers []Order, makerBook *skiplist) (err error) {
	for _, maker := range makers {
		pl := nodePayload(maker)
		makerId, _ := NewOrderId(maker.Id)
		if err = makerBook.updatePayload(makerId, &pl); err != nil {
			return err
			//fmt.Printf("failed update maker storage for err : %s\n", err.Error())
		}
		mc.emitOrderRes(maker)
	}
	size := len(makers)
	if size > 0 {
		if makers[size-1].Status == FullyExecuted || makers[size-1].Status == Cancelled {
			toTruncateId, _ := NewOrderId(makers[size-1].Id)
			if err = makerBook.truncateHeadTo(toTruncateId, int32(size)); err != nil {
				return err
			}
		} else if size >= 2 {
			toTruncateId, _ := NewOrderId(makers[size-2].Id)
			if err = makerBook.truncateHeadTo(toTruncateId, int32(size-1)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (mc *matcher) handleTakerRes(order Order) (err error) {
	if order.Status == PartialExecuted || order.Status == Pending {
		if err = mc.saveTakerAsMaker(order); err != nil {
			return err
		}
	}
	mc.emitOrderRes(order)
	return nil
}

func (mc *matcher) saveTakerAsMaker(maker Order) error {
	var (
		bookToMake *skiplist
		err error
		)
	if bookToMake, err = mc.getBookById(getBookIdToMakeForOrder(maker)); err != nil {
		return err
	}
	pl := nodePayload(maker)
	makerId, _ := NewOrderId(maker.Id)
	return bookToMake.insert(makerId, &pl)
}

func (mc *matcher) emitOrderRes(orderRes Order) {
	event := OrderUpdateEvent{orderRes.Order}
	(*mc.storage).AddLog(newLog(event))
}

func (mc *matcher) emitTxs(txs []OrderTx) {
	//fmt.Printf("matched txs >>>>>>>>> %d\n", len(txs))
	for _, tx := range txs {
		mc.handleTxSettleAction(tx)
		txEvent := TransactionEvent{tx.Transaction}
		(*mc.storage).AddLog(newLog(txEvent))
		//fmt.Printf("matched tx is : %s\n", tx.String())
	}
}

func (mc *matcher) handleRefund(order *Order) {
	if order.Status == FullyExecuted || order.Status == Cancelled {
		switch order.Side {
		case false: //buy
			order.RefundToken = order.QuoteToken
			order.RefundQuantity = SubBigInt(order.Amount, order.ExecutedAmount)
		case true:
			order.RefundToken = order.TradeToken
			order.RefundQuantity = SubBigInt(order.Quantity, order.ExecutedQuantity)
		}
		mc.updateSettleAction(proto.SettleAction{Address:order.Address, Token:order.RefundToken, ReleaseLocked:order.RefundQuantity})
	}
}

func (mc *matcher) handleTxSettleAction(tx OrderTx) {
	takerInSettle := proto.SettleAction{Address : tx.takerAddress}
	takerOutSettle := proto.SettleAction{Address : tx.takerAddress}
	makerInSettle := proto.SettleAction{Address : tx.makerAddress}
	makerOutSettle := proto.SettleAction{Address : tx.makerAddress}
	switch tx.TakerSide {
		case false: //buy
			takerInSettle.Token = tx.takerTradeToken
			takerInSettle.IncAvailable = tx.Quantity
			makerOutSettle.Token = tx.takerTradeToken
			makerOutSettle.DeduceLocked = tx.Quantity

			takerOutSettle.Token = tx.takerQuoteToken
			takerOutSettle.DeduceLocked = tx.Amount
			makerInSettle.Token = tx.takerQuoteToken
			makerInSettle.IncAvailable = tx.Amount

		case true: //sell
			takerInSettle.Token = tx.takerQuoteToken
			takerInSettle.IncAvailable = tx.Amount
			makerOutSettle.Token = tx.takerQuoteToken
			makerOutSettle.DeduceLocked = tx.Amount

			takerOutSettle.Token = tx.takerTradeToken
			takerOutSettle.DeduceLocked = tx.Quantity
			makerInSettle.Token = tx.takerTradeToken
			makerInSettle.IncAvailable = tx.Quantity
	}
	for _, ac := range []proto.SettleAction{takerInSettle, takerOutSettle, makerInSettle, makerOutSettle} {
		mc.updateSettleAction(ac)
	}
}

func (mc *matcher) updateSettleAction(action proto.SettleAction) {
	var (
		actionMap map[types.TokenTypeId]*proto.SettleAction // token -> action
		ok bool
		ac *proto.SettleAction
		address = types.Address{}
	)
	address.SetBytes(action.Address)
	if actionMap, ok = mc.settleActions[address]; !ok {
		actionMap = make(map[types.TokenTypeId]*proto.SettleAction)
	}
	token := types.TokenTypeId{}
	token.SetBytes(action.Token)
	if ac, ok = actionMap[token]; !ok {
		ac = &proto.SettleAction{Address:address.Bytes(), Token:action.Token}
	}
	ac.IncAvailable = AddBigInt(ac.IncAvailable, action.IncAvailable)
	ac.ReleaseLocked = AddBigInt(ac.ReleaseLocked, action.ReleaseLocked)
	ac.DeduceLocked = AddBigInt(ac.DeduceLocked, action.DeduceLocked)
	actionMap[token] = ac
	mc.settleActions[address] = actionMap
}


func newLog(event OrderEvent) *ledger.VmLog {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.getTopicId())
	log.Data = event.toDataBytes()
	return log
}

func matchPrice(taker Order, maker Order) (matched bool, executedPrice string) {
	if taker.Type == Market || priceEqual(taker.Price, maker.Price) {
		return true, maker.Price
	} else {
		matched = false
		tp, _ := new(big.Float).SetString(taker.Price)
		mp, _ := new(big.Float).SetString(maker.Price)
		switch taker.Side {
		case false: // buy
			matched = tp.Cmp(mp) >= 0
		case true: // sell
			matched = tp.Cmp(mp) <= 0
		}
		return matched, maker.Price
	}
}

func filterTimeout(takerTimestamp int64, maker *Order) bool {
	if takerTimestamp > maker.Timestamp+timeoutSecond {
		switch maker.Status {
		case Pending:
			maker.CancelReason = cancelledOnTimeout
		case PartialExecuted:
			maker.CancelReason = partialExecutedCancelledOnTimeout
		default:
			maker.CancelReason = unknownCancelledOnTimeout
		}
		maker.Status = Cancelled
		return true
	} else {
		return false
	}
}

func calculateOrderAndTx(taker *Order, maker *Order) (tx OrderTx) {
	tx = OrderTx{}
	tx.Id = generateTxId(taker.Id, maker.Id)
	tx.TakerSide = taker.Side
	tx.TakerId = taker.Id
	tx.MakerId = maker.Id
	tx.Price = maker.Price
	executeQuantity := MinBigInt(SubBigInt(taker.Quantity, taker.ExecutedQuantity), SubBigInt(maker.Quantity, maker.ExecutedQuantity))
	takerAmount := calculateOrderAmount(taker, executeQuantity, maker.Price)
	makerAmount := calculateOrderAmount(maker, executeQuantity, maker.Price)
	executeAmount := MinBigInt(takerAmount, makerAmount)
	executeQuantity = MinBigInt(executeQuantity, calculateQuantity(executeAmount, maker.Price))
	updateOrder(taker, executeQuantity, executeAmount)
	updateOrder(maker, executeQuantity, executeAmount)
	tx.Quantity = executeQuantity
	tx.Amount = executeAmount
	tx.Timestamp = taker.Timestamp
	tx.takerAddress = taker.Address
	tx.makerAddress = maker.Address
	tx.takerTradeToken = taker.TradeToken
	tx.takerQuoteToken = taker.QuoteToken
	tx.makerAddress = maker.Address
	return tx
}

func calculateOrderAmount(order *Order, quantity []byte, price string) []byte {
	amount := CalculateAmount(quantity, price)
	if !order.Side && new(big.Int).SetBytes(order.Amount).Cmp(new(big.Int).SetBytes(AddBigInt(order.ExecutedAmount, amount))) < 0 {// side is buy
		amount = SubBigInt(order.Amount, order.ExecutedAmount)
	}
	return amount
}

func updateOrder(order *Order, quantity []byte, amount []byte) []byte {
	order.ExecutedAmount = AddBigInt(order.ExecutedAmount, amount)
	if bytes.Equal(SubBigInt(order.Quantity, order.ExecutedQuantity), quantity) ||
		order.Type == Market && !order.Side && bytes.Equal(SubBigInt(order.Amount, order.ExecutedAmount), amount) || // market buy order
		isDust(order, quantity) {
		order.Status = FullyExecuted
	} else {
		order.Status = PartialExecuted
	}
	order.ExecutedQuantity = AddBigInt(order.ExecutedQuantity, quantity)
	return amount
}

// leave quantity is too small for calculate precision
func isDust(order *Order, quantity []byte) bool {
	return new(big.Int).SetBytes(CalculateAmount(SubBigInt(SubBigInt(order.Quantity, order.ExecutedQuantity), quantity), order.Price)).Cmp(big.NewInt(1)) < 0
}

func getMakerById(makerBook *skiplist, orderId nodeKeyType) (od Order, nextId OrderId, err error) {
	if pl, fwk, _, err := makerBook.getByKey(orderId); err != nil {
		return Order{}, (*makerBook.protocol).getNilKey().(OrderId), err
	} else {
		maker := (*pl).(Order)
		return maker, fwk.(OrderId), nil
	}
}

func ValidPrice(price string) bool {
	if len(price) == 0 {
		return false
	} else if  pr, ok := new(big.Float).SetString(price);  !ok || pr.Cmp(big.NewFloat(0)) <= 0 {
		return false
	} else {
		idx := strings.Index(price, ",")
		if idx > 0 && len(price) - idx >= 12 { // price max precision is 10 decimal
			return false
		}
	}
	return true
}

func CalculateAmount(quantity []byte, price string) []byte {
	qtF := big.NewFloat(0).SetInt(new(big.Int).SetBytes(quantity))
	prF, _ := big.NewFloat(0).SetString(price)
	amountF := prF.Mul(prF, qtF)
	amount, _ := amountF.Add(amountF, big.NewFloat(0.5)).Int(nil)
	return amount.Bytes()
}

func calculateQuantity(amount []byte, price string) []byte {
	amtF := big.NewFloat(0).SetInt(new(big.Int).SetBytes(amount))
	prF, _ := big.NewFloat(0).SetString(price)
	qtyF := amtF.Quo(amtF, prF)
	qty, _ := qtyF.Add(qtyF, big.NewFloat(0.5)).Int(nil)
	return qty.Bytes()
}

func getBookIdToTake(order Order) SkipListId {
	bytes := append(order.TradeToken, order.QuoteToken...)
	bytes = append(bytes, byte(1 - sideToInt(order.Side)))
	id := SkipListId{}
	id.SetBytes(bytes)
	return id
}

func getBookIdToMakeForOrder(order Order) SkipListId {
	return GetBookIdToMake(order.TradeToken, order.QuoteToken, order.Side)
}

func GetBookIdToMake(tradeToken []byte, quoteToken []byte, side bool) SkipListId {
	bytes := append(tradeToken, quoteToken...)
	bytes = append(bytes, byte(sideToInt(side)))
	bookId := SkipListId{}
	bookId.SetBytes(bytes)
	return bookId
}

func sideToInt(side bool) int8 {
	var sideInt int8 = 0 // buy
	if side {
		sideInt = 1 // sell
	}
	return sideInt
}

func generateTxId(takerId []byte, makerId []byte) []byte {
	return crypto.Hash(txIdLength, takerId, makerId)
}
