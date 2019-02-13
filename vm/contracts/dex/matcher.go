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
)

const maxTxsCountPerTaker = 1000
const timeoutSecond = 7 * 24 * 3600
const txIdLength = 20

type matcher struct {
	contractAddress *types.Address
	storage         *BaseStorage
	protocol        *nodePayloadProtocol
	books           map[SkipListId]*skiplist
	fundSettles     map[types.Address]map[types.TokenTypeId]*proto.FundSettle
	feeSettles      map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle
}

type OrderTx struct {
	proto.Transaction
	takerAddress    []byte
	makerAddress    []byte
	takerTradeToken []byte
	takerQuoteToken []byte
}

var (
	TakerFeeRate = 0.001
	MakerFeeRate = 0.001
)

func NewMatcher(contractAddress *types.Address, storage *BaseStorage) *matcher {
	mc := &matcher{}
	mc.contractAddress = contractAddress
	mc.storage = storage
	var po nodePayloadProtocol = &OrderNodeProtocol{}
	mc.protocol = &po
	mc.books = make(map[SkipListId]*skiplist)
	mc.fundSettles = make(map[types.Address]map[types.TokenTypeId]*proto.FundSettle)
	mc.feeSettles = make(map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle)
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

func (mc *matcher) GetFundSettles() map[types.Address]map[types.TokenTypeId]*proto.FundSettle {
	return mc.fundSettles
}

func (mc *matcher) GetFees() map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle {
	return mc.feeSettles
}

func (mc *matcher) GetOrderByIdAndBookId(orderIdBytes []byte, makerBookId SkipListId) (*Order, error) {
	var (
		book *skiplist
		err  error
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
		err  error
		ok   bool
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
			mc.handleTxs(txs)
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
			*txs = append(*txs, tx)
			if taker.Status == PartialExecuted && len(*txs) >= maxTxsCountPerTaker {
				taker.Status = Cancelled
				taker.CancelReason = partialExecutedCancelledByMarket
			}
			mc.handleRefund(taker)
			mc.handleRefund(&maker)
			*modifiedMakers = append(*modifiedMakers, maker)
		}
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

func calculateOrderAndTx(taker *Order, maker *Order) (tx OrderTx) {
	tx = OrderTx{}
	tx.Id = generateTxId(taker.Id, maker.Id)
	tx.TakerSide = taker.Side
	tx.TakerId = taker.Id
	tx.MakerId = maker.Id
	tx.Price = maker.Price
	executeQuantity := MinBigInt(SubBigIntAbs(taker.Quantity, taker.ExecutedQuantity), SubBigIntAbs(maker.Quantity, maker.ExecutedQuantity))
	takerAmount := calculateOrderAmount(taker, executeQuantity, maker.Price)
	makerAmount := calculateOrderAmount(maker, executeQuantity, maker.Price)
	executeAmount := MinBigInt(takerAmount, makerAmount)
	//fmt.Printf("calculateOrderAndTx executeQuantity %v, takerAmount %v, makerAmount %v, executeAmount %v\n", new(big.Int).SetBytes(executeQuantity).String(), new(big.Int).SetBytes(takerAmount).String(), new(big.Int).SetBytes(makerAmount).String(), new(big.Int).SetBytes(executeAmount).String())
	takerFee, takerExecutedFee := calculateFeeAndExecutedFee(taker, executeAmount, TakerFeeRate)
	makerFee, makerExecutedFee := calculateFeeAndExecutedFee(maker, executeAmount, MakerFeeRate)
	updateOrder(taker, executeQuantity, executeAmount, takerExecutedFee)
	updateOrder(maker, executeQuantity, executeAmount, makerExecutedFee)
	tx.Quantity = executeQuantity
	tx.Amount = executeAmount
	tx.takerAddress = taker.Address
	tx.makerAddress = maker.Address
	tx.takerTradeToken = taker.TradeToken
	tx.takerQuoteToken = taker.QuoteToken
	tx.makerAddress = maker.Address
	tx.TakerFee = takerFee
	tx.MakerFee = makerFee
	tx.Timestamp = taker.Timestamp
	return tx
}

func calculateOrderAmount(order *Order, quantity []byte, price string) []byte {
	amount := CalculateRawAmount(quantity, price, order.TradeTokenDecimals, order.QuoteTokenDecimals)
	if !order.Side && new(big.Int).SetBytes(order.Amount).Cmp(new(big.Int).SetBytes(AddBigInt(order.ExecutedAmount, amount))) < 0 { // side is buy
		amount = SubBigIntAbs(order.Amount, order.ExecutedAmount)
	}
	return amount
}

func updateOrder(order *Order, quantity []byte, amount []byte, executedFee []byte) []byte {
	order.ExecutedAmount = AddBigInt(order.ExecutedAmount, amount)
	if bytes.Equal(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity) ||
		order.Type == Market && !order.Side && bytes.Equal(SubBigIntAbs(order.Amount, order.ExecutedAmount), amount) || // market buy order
		isDust(order, quantity) {
		order.Status = FullyExecuted
	} else {
		order.Status = PartialExecuted
	}
	order.ExecutedFee = executedFee
	order.ExecutedQuantity = AddBigInt(order.ExecutedQuantity, quantity)
	return amount
}

// leave quantity is too small for calculate precision
func isDust(order *Order, quantity []byte) bool {
	return CalculateRawAmountF(SubBigIntAbs(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity), order.Price, order.TradeTokenDecimals, order.QuoteTokenDecimals).Cmp(new(big.Float).SetInt64(int64(1))) < 0
}

func CalculateRawAmount(quantity []byte, price string, tradeDecimals int32, quoteDecimals int32) []byte {
	return RoundAmount(CalculateRawAmountF(quantity, price, tradeDecimals, quoteDecimals)).Bytes()
}

func CalculateRawAmountF(quantity []byte, price string, tradeDecimals, quoteDecimals int32) *big.Float {
	qtF := big.NewFloat(0).SetInt(new(big.Int).SetBytes(quantity))
	prF, _ := big.NewFloat(0).SetString(price)
	amountF := new(big.Float).Mul(prF, qtF)
	return AdjustForDecimalsDiff(amountF, tradeDecimals, quoteDecimals)
}

func CalculateRawFee(amount []byte, feeRate float64) []byte {
	amtF := new(big.Float).SetInt(new(big.Int).SetBytes(amount))
	rateF := big.NewFloat(feeRate)
	amtFee := amtF.Mul(amtF, rateF)
	return RoundAmount(amtFee).Bytes()
}

func calculateFeeAndExecutedFee(order *Order, amount []byte, feeRate float64) (feeBytes, executedFee []byte) {
	feeBytes = CalculateRawFee(amount, feeRate)
	switch order.Side {
	case false: //buy
		if CmpForBigInt(order.LockedBuyFee, order.ExecutedFee) <= 0 {
			feeBytes = big.NewInt(0).Bytes()
			executedFee = order.ExecutedFee
		} else {
			executedFee = AddBigInt(order.ExecutedFee, feeBytes)
			//check if executeFee exceed lockedBuyFee
			if CmpForBigInt(order.LockedBuyFee, executedFee) <= 0 {
				feeBytes = SubBigIntAbs(order.LockedBuyFee, order.ExecutedFee)
				executedFee = order.LockedBuyFee
			}
		}

	case true: //sell
		executedFee = AddBigInt(order.ExecutedFee, feeBytes)
	}
	return feeBytes, executedFee
}


func (mc *matcher) handleRefund(order *Order) {
	if order.Status == FullyExecuted || order.Status == Cancelled {
		switch order.Side {
		case false: //buy
			order.RefundToken = order.QuoteToken
			refundAmount := SubBigIntAbs(order.Amount, order.ExecutedAmount)
			refundFee := SubBigIntAbs(order.LockedBuyFee, order.ExecutedFee)
			order.RefundQuantity = AddBigInt(refundAmount, refundFee)
		case true:
			order.RefundToken = order.TradeToken
			order.RefundQuantity = SubBigIntAbs(order.Quantity, order.ExecutedQuantity)
		}
		mc.updateFundSettle(order.Address, proto.FundSettle{Token: order.RefundToken, ReleaseLocked: order.RefundQuantity})
	}
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
		err        error
	)
	if bookToMake, err = mc.getBookById(getBookIdToMakeForOrder(maker)); err != nil {
		return err
	}
	pl := nodePayload(maker)
	makerId, _ := NewOrderId(maker.Id)
	return bookToMake.insert(makerId, &pl)
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

func (mc *matcher) emitOrderRes(orderRes Order) {
	event := OrderUpdateEvent{orderRes.Order}
	(*mc.storage).AddLog(newLog(event))
}

func (mc *matcher) handleTxs(txs []OrderTx) {
	//fmt.Printf("matched txs >>>>>>>>> %d\n", len(txs))
	for _, tx := range txs {
		mc.handleTxFundSettle(tx)
		txEvent := TransactionEvent{tx.Transaction}
		(*mc.storage).AddLog(newLog(txEvent))
		//fmt.Printf("matched tx is : %s\n", tx.String())
	}
}

func (mc *matcher) handleTxFundSettle(tx OrderTx) {
	takerInSettle := proto.FundSettle{}
	takerOutSettle := proto.FundSettle{}
	makerInSettle := proto.FundSettle{}
	makerOutSettle := proto.FundSettle{}
	switch tx.TakerSide {
	case false: //buy
		takerInSettle.Token = tx.takerTradeToken
		takerInSettle.IncAvailable = tx.Quantity
		makerOutSettle.Token = tx.takerTradeToken
		makerOutSettle.ReduceLocked = tx.Quantity

		takerOutSettle.Token = tx.takerQuoteToken
		takerOutSettle.ReduceLocked = AddBigInt(tx.Amount, tx.TakerFee)
		makerInSettle.Token = tx.takerQuoteToken
		makerInSettle.IncAvailable = SubBigIntAbs(tx.Amount, tx.MakerFee)

	case true: //sell
		takerInSettle.Token = tx.takerQuoteToken
		takerInSettle.IncAvailable = SubBigIntAbs(tx.Amount, tx.TakerFee)
		makerOutSettle.Token = tx.takerQuoteToken
		makerOutSettle.ReduceLocked = AddBigInt(tx.Amount, tx.MakerFee)

		takerOutSettle.Token = tx.takerTradeToken
		takerOutSettle.ReduceLocked = tx.Quantity
		makerInSettle.Token = tx.takerTradeToken
		makerInSettle.IncAvailable = tx.Quantity
	}
	mc.updateFundSettle(tx.takerAddress, takerInSettle)
	mc.updateFundSettle(tx.takerAddress, takerOutSettle)
	mc.updateFundSettle(tx.makerAddress, makerInSettle)
	mc.updateFundSettle(tx.makerAddress, makerOutSettle)

	mc.updateFee(tx.takerQuoteToken, tx.takerAddress, tx.TakerFee)
	mc.updateFee(tx.takerQuoteToken, tx.makerAddress, tx.MakerFee)
}

func (mc *matcher) updateFundSettle(addressBytes []byte, settle proto.FundSettle) {
	var (
		settleMap map[types.TokenTypeId]*proto.FundSettle // token -> settle
		ok        bool
		ac        *proto.FundSettle
		address   = types.Address{}
	)
	address.SetBytes(addressBytes)
	if settleMap, ok = mc.fundSettles[address]; !ok {
		settleMap = make(map[types.TokenTypeId]*proto.FundSettle)
		mc.fundSettles[address] = settleMap
	}
	token := types.TokenTypeId{}
	token.SetBytes(settle.Token)
	if ac, ok = settleMap[token]; !ok {
		ac = &proto.FundSettle{Token: settle.Token}
		settleMap[token] = ac
	}
	ac.IncAvailable = AddBigInt(ac.IncAvailable, settle.IncAvailable)
	ac.ReleaseLocked = AddBigInt(ac.ReleaseLocked, settle.ReleaseLocked)
	ac.ReduceLocked = AddBigInt(ac.ReduceLocked, settle.ReduceLocked)
}

func (mc *matcher) updateFee(quoteToken []byte, address []byte, amount []byte) {
	var (
		userFeeSettles map[types.Address]*proto.UserFeeSettle
		userFeeSettle  *proto.UserFeeSettle
		ok             bool
	)

	token := types.TokenTypeId{}
	token.SetBytes(quoteToken)
	if userFeeSettles, ok = mc.feeSettles[token]; !ok {
		userFeeSettles = make(map[types.Address]*proto.UserFeeSettle)
		mc.feeSettles[token] = userFeeSettles
	}
	addr := types.Address{}
	addr.SetBytes(address)
	if userFeeSettle, ok = userFeeSettles[addr]; !ok {
		userFeeSettle = &proto.UserFeeSettle{Address: address, Amount:amount}
		userFeeSettles[addr] = userFeeSettle
	} else {
		userFeeSettle.Amount = AddBigInt(userFeeSettle.Amount, amount)
	}
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

func getMakerById(makerBook *skiplist, orderId nodeKeyType) (od Order, nextId OrderId, err error) {
	if pl, fwk, _, err := makerBook.getByKey(orderId); err != nil {
		return Order{}, (*makerBook.protocol).getNilKey().(OrderId), err
	} else {
		maker := (*pl).(Order)
		return maker, fwk.(OrderId), nil
	}
}

func getBookIdToTake(order Order) SkipListId {
	bytes := append(order.TradeToken, order.QuoteToken...)
	bytes = append(bytes, byte(1-sideToInt(order.Side)))
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

func MaxFeeRate() float64 {
	if TakerFeeRate > MakerFeeRate {
		return TakerFeeRate
	} else {
		return MakerFeeRate
	}
}

// only for unit test
func SetFeeRate(takerFR float64, makerFR float64) {
	TakerFeeRate = takerFR
	MakerFeeRate = makerFR
}
