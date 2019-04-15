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
const bigFloatPrec = 120

type Matcher struct {
	contractAddress *types.Address
	storage         *BaseStorage
	protocol        *nodePayloadProtocol
	books           map[SkipListId]*skiplist
	fundSettles     map[types.Address]map[types.TokenTypeId]*proto.FundSettle
	feeSettles      map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle
}

type OrderTx struct {
	proto.Transaction
	takerAddress []byte
	makerAddress []byte
	tradeToken   []byte
	quoteToken   []byte
}

var (
	TakerFeeRate          = "0.0025"
	MakerFeeRate          = "0.0025"
	DeleteTerminatedOrder = true
)

func NewMatcher(contractAddress *types.Address, storage *BaseStorage) *Matcher {
	mc := &Matcher{}
	mc.contractAddress = contractAddress
	mc.storage = storage
	var po nodePayloadProtocol = &OrderNodeProtocol{}
	mc.protocol = &po
	mc.books = make(map[SkipListId]*skiplist)
	mc.fundSettles = make(map[types.Address]map[types.TokenTypeId]*proto.FundSettle)
	mc.feeSettles = make(map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle)
	return mc
}

func (mc *Matcher) MatchOrder(taker TakerOrder) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("matchOrder failed with exception %v", r)
		}
	}()
	if mc.checkOrderIdExists(taker.Order.Id) {
		return fmt.Errorf("order id already exists")
	}
	var bookToTake *skiplist
	if bookToTake, err = mc.getBookById(getBookIdToTake(taker)); err != nil {
		return err
	}
	if err := mc.doMatchTaker(taker, bookToTake); err != nil {
		return err
	}
	return nil
}

func (mc *Matcher) GetFundSettles() map[types.Address]map[types.TokenTypeId]*proto.FundSettle {
	return mc.fundSettles
}

func (mc *Matcher) GetFees() map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle {
	return mc.feeSettles
}

func (mc *Matcher) GetOrderByIdAndBookId(makerBookId SkipListId, orderIdBytes []byte) (*Order, error) {
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
		return nil, GetOrderByIdFailedErr
	} else {
		od, _ := (*pl).(Order)
		return &od, nil
	}
}

func (mc *Matcher) GetOrdersFromMarket(makerBookId SkipListId, begin, end int32) ([]*Order, int32, error) {
	var (
		book *skiplist
		err  error
		i int32 = 0
		pl *nodePayload
		forward nodeKeyType
	)
	if begin >= end {
		return nil, book.length, nil
	}
	if book, err = mc.getBookById(makerBookId); err != nil || book.length == 0 {
		return nil, 0, err
	}
	orders := make([]*Order, 0, end - begin)
	currentId := book.header
	for ; i < book.length && i < end; i++ {
		if pl, forward, _, err = book.getByKey(currentId); err != nil {
			//fmt.Printf("getOrderFailed index %d, book.length %d, currentId %s, err %s\n", i, book.length, currentId.toString(), err.Error())
			return nil, book.length, err
		} else if i >= begin {
			//fmt.Printf("index %d, currentId %s, forward %s\n", i, currentId.toString(), forward.toString())
			od, _ := (*pl).(Order)
 			orders = append(orders, &od)
			currentId = forward
		} else {
			currentId = forward
		}
	}
	return orders, book.length, nil
}

func (mc *Matcher) CancelOrderByIdAndBookId(order *Order, makerBookId SkipListId, tradeToken, quoteToken types.TokenTypeId) (err error) {
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
	orderTokenInfo := &proto.OrderTokenInfo{TradeToken:tradeToken.Bytes(), QuoteToken:quoteToken.Bytes()}
	mc.handleRefund(&order.Order, orderTokenInfo)
	mc.emitOrderUpdate(*order, orderTokenInfo)
	pl := nodePayload(*order)
	var orderId OrderId
	if orderId, err = NewOrderId(order.Id); err != nil {
		return err
	} else if !orderId.IsNormal() {
		return fmt.Errorf("invalid order id format")
	}
	if err = book.delete(orderId); err != nil {
		return err
	}
	if DeleteTerminatedOrder {
		mc.rawDelete(orderId)
		return nil
	} else {
		return book.updatePayload(orderId, &pl)
	}
}

func (mc *Matcher) getBookById(bookId SkipListId) (*skiplist, error) {
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

func (mc *Matcher) doMatchTaker(taker TakerOrder, makerBook *skiplist) (err error) {
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
			if err = mc.handleModifiedMakers(modifiedMakers, makerBook, taker.OrderTokenInfo); err != nil {
				return err
			}
			mc.handleTxs(txs)
			return nil
		}
	}
}

//TODO add assertion for order calculation correctness
func (mc *Matcher) recursiveTakeOrder(taker *TakerOrder, maker Order, makerBook *skiplist, modifiedMakers *[]Order, txs *[]OrderTx, nextOrderId nodeKeyType) error {
	if filterTimeout(taker.Order.Timestamp, &maker) {
		mc.handleRefund(&maker.Order, taker.OrderTokenInfo)
		*modifiedMakers = append(*modifiedMakers, maker)
	} else {
		matched, _ := matchPrice(*taker, maker)
		//fmt.Printf("recursiveTakeOrder matched for taker.id %d is %t\n", taker.Id, matched)
		if matched {
			tx := calculateOrderAndTx(taker, &maker)
			*txs = append(*txs, tx)
			if taker.Order.Status == PartialExecuted && len(*txs) >= maxTxsCountPerTaker {
				taker.Order.Status = Cancelled
				taker.Order.CancelReason = partialExecutedCancelledByMarket
			}
			mc.handleRefund(taker.Order, taker.OrderTokenInfo)
			mc.handleRefund(&maker.Order, taker.OrderTokenInfo)
			*modifiedMakers = append(*modifiedMakers, maker)
		}
	}
	if taker.Order.Status == FullyExecuted || taker.Order.Status == Cancelled {
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

func calculateOrderAndTx(taker *TakerOrder, maker *Order) (tx OrderTx) {
	tx = OrderTx{}
	tx.Id = generateTxId(taker.Order.Id, maker.Id)
	tx.TakerSide = taker.Order.Side
	tx.TakerId = taker.Order.Id
	tx.MakerId = maker.Id
	tx.Price = maker.Price
	executeQuantity := MinBigInt(SubBigIntAbs(taker.Order.Quantity, taker.Order.ExecutedQuantity), SubBigIntAbs(maker.Quantity, maker.ExecutedQuantity))
	takerAmount := calculateOrderAmount(taker.Order, executeQuantity, maker.Price, taker.OrderTokenInfo)
	makerAmount := calculateOrderAmount(&maker.Order, executeQuantity, maker.Price, taker.OrderTokenInfo)
	executeAmount := MinBigInt(takerAmount, makerAmount)
	//fmt.Printf("calculateOrderAndTx executeQuantity %v, takerAmount %v, makerAmount %v, executeAmount %v\n", new(big.Int).SetBytes(executeQuantity).String(), new(big.Int).SetBytes(takerAmount).String(), new(big.Int).SetBytes(makerAmount).String(), new(big.Int).SetBytes(executeAmount).String())
	takerFee, takerExecutedFee := calculateFeeAndExecutedFee(taker.Order, executeAmount, TakerFeeRate)
	makerFee, makerExecutedFee := calculateFeeAndExecutedFee(&maker.Order, executeAmount, MakerFeeRate)
	updateOrder(taker.Order, executeQuantity, executeAmount, takerExecutedFee, taker.OrderTokenInfo)
	updateOrder(&maker.Order, executeQuantity, executeAmount, makerExecutedFee, taker.OrderTokenInfo)
	tx.Quantity = executeQuantity
	tx.Amount = executeAmount
	tx.takerAddress = taker.Order.Address
	tx.makerAddress = maker.Address
	tx.tradeToken = taker.OrderTokenInfo.TradeToken
	tx.quoteToken = taker.OrderTokenInfo.QuoteToken
	tx.makerAddress = maker.Address
	tx.TakerFee = takerFee
	tx.MakerFee = makerFee
	tx.Timestamp = taker.Order.Timestamp
	return tx
}

func calculateOrderAmount(order *proto.Order, quantity []byte, price string, tokenInfo *proto.OrderTokenInfo) []byte {
	amount := CalculateRawAmount(quantity, price, tokenInfo.TradeTokenDecimals, tokenInfo.QuoteTokenDecimals)
	if !order.Side && new(big.Int).SetBytes(order.Amount).Cmp(new(big.Int).SetBytes(AddBigInt(order.ExecutedAmount, amount))) < 0 { // side is buy
		amount = SubBigIntAbs(order.Amount, order.ExecutedAmount)
	}
	return amount
}

func updateOrder(order *proto.Order, quantity []byte, amount []byte, executedFee []byte, tokenInfo *proto.OrderTokenInfo) []byte {
	order.ExecutedAmount = AddBigInt(order.ExecutedAmount, amount)
	if bytes.Equal(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity) ||
		order.Type == Market && !order.Side && bytes.Equal(SubBigIntAbs(order.Amount, order.ExecutedAmount), amount) || // market buy order
		isDust(order, quantity, tokenInfo) {
		order.Status = FullyExecuted
	} else {
		order.Status = PartialExecuted
	}
	order.ExecutedFee = executedFee
	order.ExecutedQuantity = AddBigInt(order.ExecutedQuantity, quantity)
	return amount
}

// leave quantity is too small for calculate precision
func isDust(order *proto.Order, quantity []byte, tokenInfo *proto.OrderTokenInfo) bool {
	return CalculateRawAmountF(SubBigIntAbs(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity), order.Price, tokenInfo.TradeTokenDecimals, tokenInfo.QuoteTokenDecimals).Cmp(new(big.Float).SetInt64(int64(1))) < 0
}

func CalculateRawAmount(quantity []byte, price string, tradeDecimals int32, quoteDecimals int32) []byte {
	return RoundAmount(CalculateRawAmountF(quantity, price, tradeDecimals, quoteDecimals)).Bytes()
}

func CalculateRawAmountF(quantity []byte, price string, tradeDecimals, quoteDecimals int32) *big.Float {
	qtF := new(big.Float).SetPrec(bigFloatPrec).SetInt(new(big.Int).SetBytes(quantity))
	prF, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(price)
	amountF := new(big.Float).SetPrec(bigFloatPrec).Mul(prF, qtF)
	return AdjustForDecimalsDiff(amountF, tradeDecimals, quoteDecimals)
}

func CalculateRawFee(amount []byte, feeRate string) []byte {
	amtF := new(big.Float).SetPrec(bigFloatPrec).SetInt(new(big.Int).SetBytes(amount))
	rateF, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(feeRate)
	amtFee := amtF.Mul(amtF, rateF)
	return RoundAmount(amtFee).Bytes()
}

func calculateFeeAndExecutedFee(order *proto.Order, amount []byte, feeRate string) (feeBytes, executedFee []byte) {
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

func (mc *Matcher) handleRefund(order *proto.Order, orderTokenInfo *proto.OrderTokenInfo) {
	if order.Status == FullyExecuted || order.Status == Cancelled {
		switch order.Side {
		case false: //buy
			order.RefundToken = orderTokenInfo.QuoteToken
			refundAmount := SubBigIntAbs(order.Amount, order.ExecutedAmount)
			refundFee := SubBigIntAbs(order.LockedBuyFee, order.ExecutedFee)
			order.RefundQuantity = AddBigInt(refundAmount, refundFee)
		case true:
			order.RefundToken = orderTokenInfo.TradeToken
			order.RefundQuantity = SubBigIntAbs(order.Quantity, order.ExecutedQuantity)
		}
		mc.updateFundSettle(order.Address, proto.FundSettle{Token: order.RefundToken, ReleaseLocked: order.RefundQuantity})
	}
}

func (mc *Matcher) handleTakerRes(taker TakerOrder) error {
	if taker.Order.Status == PartialExecuted || taker.Order.Status == Pending {
		if bookToMake, err := mc.getBookById(getBookIdToMakeForTaker(taker)); err != nil {
			return err
		} else if err = mc.saveTakerAsMaker(taker, bookToMake); err != nil {
			return err
		}
	}
	mc.emitNewOrder(taker)
	return nil
}

func (mc *Matcher) saveTakerAsMaker(taker TakerOrder, bookToMake *skiplist) error {
	order := Order{*taker.Order}
	pl := nodePayload(order)
	makerId, _ := NewOrderId(taker.Order.Id)
	return bookToMake.insert(makerId, &pl)
}

func (mc *Matcher) checkOrderIdExists(orderIdBytes []byte) bool {
	orderId, _ := NewOrderId(orderIdBytes)
	if len((*mc.storage).GetStorage(mc.contractAddress, orderId.getStorageKey())) == 0 {
		return false
	} else {
		return true
	}
}

func (mc *Matcher) handleModifiedMakers(makers []Order, makerBook *skiplist, tokenInfo *proto.OrderTokenInfo) (err error) {
	size := len(makers)
	var truncatedSize int // will be zero in case only one partiallyExecuted order
	if size > 0 {
		if makers[size-1].Status == FullyExecuted || makers[size-1].Status == Cancelled {
			toTruncateId, _ := NewOrderId(makers[size-1].Id)
			if err = makerBook.truncateHeadTo(toTruncateId, int32(size)); err != nil {
				return err
			}
			truncatedSize = size
		} else if size >= 2 {
			toTruncateId, _ := NewOrderId(makers[size-2].Id)
			if err = makerBook.truncateHeadTo(toTruncateId, int32(size-1)); err != nil {
				return err
			}
			truncatedSize = size - 1
		}
	}

	for index, maker := range makers {
		makerId, _ := NewOrderId(maker.Id)
		if DeleteTerminatedOrder && index <= truncatedSize - 1 {
			mc.rawDelete(makerId)
		} else {
			pl := nodePayload(maker)
			if err = makerBook.updatePayload(makerId, &pl); err != nil {
				return err
				//fmt.Printf("failed update maker storage for err : %s\n", err.Error())
			}
		}
		mc.emitOrderUpdate(maker, tokenInfo)
	}
	return nil
}

func (mc *Matcher) emitNewOrder(taker TakerOrder) {
	event := NewOrderEvent{taker.OrderInfo}
	(*mc.storage).AddLog(newLog(event))
}

func (mc *Matcher) emitOrderUpdate(order Order, tokenInfo *proto.OrderTokenInfo) {
	updateInfo := proto.OrderUpdateInfo{}
	updateInfo.Id = order.Id
	updateInfo.Status = order.Status
	updateInfo.CancelReason = order.CancelReason
	updateInfo.ExecutedQuantity = order.ExecutedQuantity
	updateInfo.ExecutedAmount = order.ExecutedAmount
	updateInfo.ExecutedFee = order.ExecutedFee
	updateInfo.RefundToken = order.RefundToken
	updateInfo.RefundQuantity = order.RefundQuantity
	updateInfo.OrderTokenInfo = tokenInfo
	event := OrderUpdateEvent{updateInfo}
	(*mc.storage).AddLog(newLog(event))
}

func (mc *Matcher) handleTxs(txs []OrderTx) {
	//fmt.Printf("matched txs >>>>>>>>> %d\n", len(txs))
	for _, tx := range txs {
		mc.handleTxFundSettle(tx)
		txEvent := TransactionEvent{tx.Transaction}
		(*mc.storage).AddLog(newLog(txEvent))
		//fmt.Printf("matched tx is : %s\n", tx.String())
	}
}

func (mc *Matcher) handleTxFundSettle(tx OrderTx) {
	takerInSettle := proto.FundSettle{}
	takerOutSettle := proto.FundSettle{}
	makerInSettle := proto.FundSettle{}
	makerOutSettle := proto.FundSettle{}
	switch tx.TakerSide {
	case false: //buy
		takerInSettle.Token = tx.tradeToken
		takerInSettle.IncAvailable = tx.Quantity
		makerOutSettle.Token = tx.tradeToken
		makerOutSettle.ReduceLocked = tx.Quantity

		takerOutSettle.Token = tx.quoteToken
		takerOutSettle.ReduceLocked = AddBigInt(tx.Amount, tx.TakerFee)
		makerInSettle.Token = tx.quoteToken
		makerInSettle.IncAvailable = SubBigIntAbs(tx.Amount, tx.MakerFee)

	case true: //sell
		takerInSettle.Token = tx.quoteToken
		takerInSettle.IncAvailable = SubBigIntAbs(tx.Amount, tx.TakerFee)
		makerOutSettle.Token = tx.quoteToken
		makerOutSettle.ReduceLocked = AddBigInt(tx.Amount, tx.MakerFee)

		takerOutSettle.Token = tx.tradeToken
		takerOutSettle.ReduceLocked = tx.Quantity
		makerInSettle.Token = tx.tradeToken
		makerInSettle.IncAvailable = tx.Quantity
	}
	mc.updateFundSettle(tx.takerAddress, takerInSettle)
	mc.updateFundSettle(tx.takerAddress, takerOutSettle)
	mc.updateFundSettle(tx.makerAddress, makerInSettle)
	mc.updateFundSettle(tx.makerAddress, makerOutSettle)

	mc.updateFee(tx.quoteToken, tx.takerAddress, tx.TakerFee)
	mc.updateFee(tx.quoteToken, tx.makerAddress, tx.MakerFee)
}

func (mc *Matcher) updateFundSettle(addressBytes []byte, settle proto.FundSettle) {
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

func (mc *Matcher) updateFee(quoteToken []byte, address []byte, amount []byte) {
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
		userFeeSettle = &proto.UserFeeSettle{Address: address, Amount: amount}
		userFeeSettles[addr] = userFeeSettle
	} else {
		userFeeSettle.Amount = AddBigInt(userFeeSettle.Amount, amount)
	}
}

func (mc *Matcher) rawDelete(orderId OrderId) {
	(*mc.storage).SetStorage(orderId.getStorageKey(), nil)
}

func newLog(event OrderEvent) *ledger.VmLog {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	return log
}

func matchPrice(taker TakerOrder, maker Order) (matched bool, executedPrice string) {
	if taker.Order.Type == Market || priceEqual(taker.Order.Price, maker.Price) {
		return true, maker.Price
	} else {
		matched = false
		tp, _ := new(big.Float).SetString(taker.Order.Price)
		mp, _ := new(big.Float).SetString(maker.Price)
		switch taker.Order.Side {
		case false: // buy
			matched = tp.Cmp(mp) >= 0
		case true: // sell
			matched = tp.Cmp(mp) <= 0
		}
		return matched, maker.Price
	}
}


//TODO support timeout when timer trigger available for set raw timestamp in one hour for order timestamp
func filterTimeout(takerTimestamp int64, maker *Order) bool {
	return false
	if takerTimestamp > maker.Timestamp + timeoutSecond {
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

func getBookIdToTake(takerOrder TakerOrder) SkipListId {
	bytes := append(takerOrder.OrderTokenInfo.TradeToken, takerOrder.OrderTokenInfo.QuoteToken...)
	bytes = append(bytes, byte(1-sideToInt(takerOrder.Order.Side)))
	id := SkipListId{}
	id.SetBytes(bytes)
	return id
}

func getBookIdToMakeForTaker(taker TakerOrder) SkipListId {
	return GetBookIdToMake(taker.OrderTokenInfo.TradeToken, taker.OrderTokenInfo.QuoteToken, taker.Order.Side)
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

func MaxFeeRate() string {
	if TakerFeeRate > MakerFeeRate {
		return TakerFeeRate
	} else {
		return MakerFeeRate
	}
}

// only for unit test
func SetFeeRate(takerFR string, makerFR string) {
	TakerFeeRate = takerFR
	MakerFeeRate = makerFR
}
