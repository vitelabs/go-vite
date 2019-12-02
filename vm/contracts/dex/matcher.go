package dex

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

const maxTxsCountPerTaker = 100
const timeoutSecond = 30 * 24 * 3600
const txIdLength = 20
const bigFloatPrec = 120

type Matcher struct {
	db          vm_db.VmDb
	MarketInfo  *MarketInfo
	fundSettles map[types.Address]map[bool]*proto.AccountSettle
	feeSettles  map[types.Address]*proto.FeeSettle
}

type OrderTx struct {
	proto.Transaction
	takerAddress []byte
	makerAddress []byte
	tradeToken   []byte
	quoteToken   []byte
}

var (
	BaseFeeRate        int32 = 200 // 200/100,000 = 0.002
	VipReduceFeeRate   int32 = 100 // 0.001
	MaxOperatorFeeRate int32 = 200 // 0.002

	PerPeriodDividendRate int32 = 1000 // 0.01

	InviterBonusRate int32 = 5000 // 0.05
	InviteeBonusRate int32 = 2500 // 0.025

	RateCardinalNum int32 = 100000 // 100,000
)

func NewMatcher(db vm_db.VmDb, marketId int32) (mc *Matcher, err error) {
	mc = NewRawMatcher(db)
	var ok bool
	if mc.MarketInfo, ok = GetMarketInfoById(db, marketId); !ok {
		return nil, TradeMarketNotExistsErr
	}
	return
}

func NewMatcherWithMarketInfo(db vm_db.VmDb, marketInfo *MarketInfo) (mc *Matcher) {
	mc = NewRawMatcher(db)
	mc.MarketInfo = marketInfo
	return
}

func NewRawMatcher(db vm_db.VmDb) (mc *Matcher) {
	mc = &Matcher{}
	mc.db = db
	mc.fundSettles = make(map[types.Address]map[bool]*proto.AccountSettle)
	mc.feeSettles = make(map[types.Address]*proto.FeeSettle)
	return
}

func (mc *Matcher) MatchOrder(taker *Order, preHash types.Hash) (err error) {
	var bookToTake *levelDbBook
	if bookToTake, err = mc.getOrderBookForTaker(taker.Side); err != nil {
		return err
	} else {
		defer bookToTake.release()
	}
	if err := mc.doMatchTaker(taker, bookToTake, preHash); err != nil {
		return err
	}
	return nil
}

func (mc *Matcher) GetFundSettles() map[types.Address]map[bool]*proto.AccountSettle {
	return mc.fundSettles
}

func (mc *Matcher) GetFees() map[types.Address]*proto.FeeSettle {
	return mc.feeSettles
}

func (mc *Matcher) GetOrderById(orderId []byte) (*Order, error) {
	if data := getValueFromDb(mc.db, orderId); len(data) > 0 {
		order := &Order{}
		if err := order.DeSerializeCompact(data, orderId); err != nil {
			return nil, err
		} else {
			return order, nil
		}
	} else {
		return nil, OrderNotExistsErr
	}
}

func (mc *Matcher) GetOrdersFromMarket(side bool, begin, end int) ([]*Order, int, error) {
	var (
		book *levelDbBook
		err  error
	)
	if begin >= end {
		return nil, 0, nil
	}
	if book, err = getMakerBook(mc.db, mc.MarketInfo.MarketId, side); err != nil {
		return nil, 0, err
	} else {
		defer book.release()
	}
	orders := make([]*Order, 0, end-begin)
	for i := 0; i < end; i++ {
		if order, ok := book.nextOrder(); !ok {
			return orders, len(orders), nil
		} else {
			if i >= begin {
				orders = append(orders, order)
			}
		}
	}
	return orders, len(orders), nil
}

func (mc *Matcher) CancelOrderById(order *Order) {
	switch order.Status {
	case Pending:
		order.CancelReason = cancelledByUser
	case PartialExecuted:
		order.CancelReason = partialExecutedUserCancelled
	}
	order.Status = Cancelled
	mc.handleRefund(order)
	mc.emitOrderUpdate(*order)
	mc.deleteOrder(order)
}

func (mc *Matcher) doMatchTaker(taker *Order, makerBook *levelDbBook, preHash types.Hash) (err error) {
	modifiedMakers := make([]*Order, 0, 20)
	txs := make([]*OrderTx, 0, 20)
	if maker, ok := makerBook.nextOrder(); !ok {
		mc.handleTakerRes(taker)
	} else {
		// must not set db in recursiveTakeOrder
		if err = mc.recursiveTakeOrder(taker, maker, makerBook, &modifiedMakers, &txs, IsDexFeeFork(mc.db)); err != nil {
			return
		} else {
			mc.handleTakerRes(taker)
			mc.handleModifiedMakers(modifiedMakers)
			mc.handleTxs(txs)
		}
	}
	TryUpdateTimestamp(mc.db, taker.Timestamp, preHash)
	return
}

//TODO add assertion for order calculation correctness
func (mc *Matcher) recursiveTakeOrder(taker, maker *Order, makerBook *levelDbBook, modifiedMakers *[]*Order, txs *[]*OrderTx, isDexFeeFork bool) error {
	if filterTimeout(mc.db, maker) {
		*modifiedMakers = append(*modifiedMakers, maker)
	} else {
		matched, _ := matchPrice(taker, maker)
		//fmt.Printf("recursiveTakeOrder matched for taker.id %d is %t\n", taker.Id, matched)
		if matched {
			tx := calculateOrderAndTx(taker, maker, mc.MarketInfo, isDexFeeFork)
			*txs = append(*txs, tx)
			if taker.Status == PartialExecuted && len(*txs) >= maxTxsCountPerTaker {
				taker.Status = Cancelled
				taker.CancelReason = partialExecutedCancelledByMarket
			}
			*modifiedMakers = append(*modifiedMakers, maker)
		} else {
			return nil
		}
	}
	if taker.Status == FullyExecuted || taker.Status == Cancelled {
		return nil
	}
	if newMaker, ok := makerBook.nextOrder(); ok {
		return mc.recursiveTakeOrder(taker, newMaker, makerBook, modifiedMakers, txs, isDexFeeFork)
	} else {
		return nil
	}
}

func (mc *Matcher) handleTakerRes(taker *Order) {
	if taker.Status == PartialExecuted || taker.Status == Pending {
		mc.saveOrder(*taker, true)
	} else { // in case isDust still need refund FullExecuted status order
		mc.handleRefund(taker)
	}
	mc.emitNewOrder(*taker)
}

func (mc *Matcher) handleModifiedMakers(makers []*Order) {
	for _, maker := range makers {
		if maker.Status == FullyExecuted || maker.Status == Cancelled {
			mc.handleRefund(maker) // in case isDust still need refund FullExecuted status order
			mc.deleteOrder(maker)
		} else {
			mc.saveOrder(*maker, false)
		}
		mc.emitOrderUpdate(*maker)
	}
}

func (mc *Matcher) handleRefund(order *Order) {
	if order.Status == FullyExecuted || order.Status == Cancelled {
		switch order.Side {
		case false: //buy
			order.RefundToken = mc.MarketInfo.QuoteToken
			refundAmount := SubBigIntAbs(order.Amount, order.ExecutedAmount)
			refundFee := SubBigIntAbs(SubBigIntAbs(order.LockedBuyFee, order.ExecutedBaseFee), order.ExecutedOperatorFee)
			order.RefundQuantity = AddBigInt(refundAmount, refundFee)
		case true:
			order.RefundToken = mc.MarketInfo.TradeToken
			order.RefundQuantity = SubBigIntAbs(order.Quantity, order.ExecutedQuantity)
		}
		if CmpToBigZero(order.RefundQuantity) > 0 {
			mc.updateFundSettle(order.Address, proto.AccountSettle{IsTradeToken: order.Side, ReleaseLocked: order.RefundQuantity})
		} else {
			order.RefundToken = nil
			order.RefundQuantity = nil
		}
	}
}

func (mc *Matcher) emitNewOrder(taker Order) {
	newOrderInfo := proto.NewOrderInfo{}
	newOrderInfo.Order = &taker.Order
	newOrderInfo.TradeToken = mc.MarketInfo.TradeToken
	newOrderInfo.QuoteToken = mc.MarketInfo.QuoteToken
	event := NewOrderEvent{newOrderInfo}
	(mc.db).AddLog(newLog(event))
}

func (mc *Matcher) emitOrderUpdate(order Order) {
	updateInfo := proto.OrderUpdateInfo{}
	updateInfo.Id = order.Id
	updateInfo.TradeToken = mc.MarketInfo.TradeToken
	updateInfo.QuoteToken = mc.MarketInfo.QuoteToken
	updateInfo.Status = order.Status
	updateInfo.CancelReason = order.CancelReason
	updateInfo.ExecutedQuantity = order.ExecutedQuantity
	updateInfo.ExecutedAmount = order.ExecutedAmount
	updateInfo.ExecutedBaseFee = order.ExecutedBaseFee
	updateInfo.ExecutedOperatorFee = order.ExecutedOperatorFee
	updateInfo.RefundToken = order.RefundToken
	updateInfo.RefundQuantity = order.RefundQuantity
	event := OrderUpdateEvent{updateInfo}
	(mc.db).AddLog(newLog(event))
}

func (mc *Matcher) handleTxs(txs []*OrderTx) {
	//fmt.Printf("matched txs >>>>>>>>> %d\n", len(txs))
	for _, tx := range txs {
		mc.handleTxFundSettle(*tx)
		txEvent := TransactionEvent{tx.Transaction}
		mc.db.AddLog(newLog(txEvent))
		//fmt.Printf("matched tx is : %s\n", tx.String())
	}
}

func (mc *Matcher) handleTxFundSettle(tx OrderTx) {
	takerInSettle := proto.AccountSettle{}
	takerOutSettle := proto.AccountSettle{}
	makerInSettle := proto.AccountSettle{}
	makerOutSettle := proto.AccountSettle{}
	switch tx.TakerSide {
	case false: //buy
		takerInSettle.IsTradeToken = true
		takerInSettle.IncAvailable = tx.Quantity
		makerOutSettle.IsTradeToken = true
		makerOutSettle.ReduceLocked = tx.Quantity

		takerOutSettle.IsTradeToken = false
		takerOutSettle.ReduceLocked = AddBigInt(tx.Amount, AddBigInt(tx.TakerFee, tx.TakerOperatorFee))
		makerInSettle.IsTradeToken = false
		makerInSettle.IncAvailable = SubBigIntAbs(tx.Amount, AddBigInt(tx.MakerFee, tx.MakerOperatorFee))

	case true: //sell
		takerInSettle.IsTradeToken = false
		takerInSettle.IncAvailable = SubBigIntAbs(tx.Amount, AddBigInt(tx.TakerFee, tx.TakerOperatorFee))
		makerOutSettle.IsTradeToken = false
		makerOutSettle.ReduceLocked = AddBigInt(tx.Amount, AddBigInt(tx.MakerFee, tx.MakerOperatorFee))

		takerOutSettle.IsTradeToken = true
		takerOutSettle.ReduceLocked = tx.Quantity
		makerInSettle.IsTradeToken = true
		makerInSettle.IncAvailable = tx.Quantity
	}
	mc.updateFundSettle(tx.takerAddress, takerInSettle)
	mc.updateFundSettle(tx.takerAddress, takerOutSettle)
	mc.updateFundSettle(tx.makerAddress, makerInSettle)
	mc.updateFundSettle(tx.makerAddress, makerOutSettle)

	mc.updateFee(tx.takerAddress, tx.TakerFee, tx.TakerOperatorFee)
	mc.updateFee(tx.makerAddress, tx.MakerFee, tx.MakerOperatorFee)
}

func (mc *Matcher) updateFundSettle(addressBytes []byte, settle proto.AccountSettle) {
	var (
		settleMap map[bool]*proto.AccountSettle // token -> settle
		ok        bool
		ac        *proto.AccountSettle
		address   = types.Address{}
	)
	address.SetBytes(addressBytes)
	if settleMap, ok = mc.fundSettles[address]; !ok {
		settleMap = make(map[bool]*proto.AccountSettle)
		mc.fundSettles[address] = settleMap
	}
	if ac, ok = settleMap[settle.IsTradeToken]; !ok {
		ac = &proto.AccountSettle{IsTradeToken: settle.IsTradeToken}
		settleMap[settle.IsTradeToken] = ac
	}
	ac.IncAvailable = AddBigInt(ac.IncAvailable, settle.IncAvailable)
	ac.ReleaseLocked = AddBigInt(ac.ReleaseLocked, settle.ReleaseLocked)
	ac.ReduceLocked = AddBigInt(ac.ReduceLocked, settle.ReduceLocked)
}

func (mc *Matcher) updateFee(address []byte, feeAmt, operatorFee []byte) {
	var (
		feeSettle *proto.FeeSettle
		ok        bool
	)
	addr := types.Address{}
	addr.SetBytes(address)
	if feeSettle, ok = mc.feeSettles[addr]; !ok {
		feeSettle = &proto.FeeSettle{Address: address, BaseFee: feeAmt, OperatorFee: operatorFee}
		mc.feeSettles[addr] = feeSettle
	} else {
		feeSettle.BaseFee = AddBigInt(feeSettle.BaseFee, feeAmt)
		feeSettle.OperatorFee = AddBigInt(feeSettle.OperatorFee, operatorFee)
	}
}

func (mc *Matcher) getOrderBookForTaker(takerSide bool) (*levelDbBook, error) {
	return getMakerBook(mc.db, mc.MarketInfo.MarketId, !takerSide)
}

func (mc *Matcher) saveOrder(order Order, isTaker bool) {
	orderId := order.Id
	if data, err := order.SerializeCompact(); err != nil {
		panic(err)
	} else {
		setValueToDb(mc.db, orderId, data)
	}
	if isTaker && len(order.SendHash) > 0 {
		SaveHashMapOrderId(mc.db, order.SendHash, orderId)
	}
}

func (mc *Matcher) deleteOrder(order *Order) {
	setValueToDb(mc.db, order.Id, nil)
	if len(order.SendHash) > 0 {
		DeleteHashMapOrderId(mc.db, order.SendHash)
	}
}

func calculateOrderAndTx(taker, maker *Order, marketInfo *MarketInfo, isDexFeeFork bool) (tx *OrderTx) {
	tx = &OrderTx{}
	tx.Id = generateTxId(taker.Id, maker.Id)
	tx.TakerSide = taker.Side
	tx.TakerId = taker.Id
	tx.MakerId = maker.Id
	tx.Price = maker.Price
	executeQuantity := MinBigInt(SubBigIntAbs(taker.Quantity, taker.ExecutedQuantity), SubBigIntAbs(maker.Quantity, maker.ExecutedQuantity))
	takerAmount := calculateOrderAmount(taker, executeQuantity, maker.Price, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
	makerAmount := calculateOrderAmount(maker, executeQuantity, maker.Price, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
	executeAmount := MinBigInt(takerAmount, makerAmount)
	//fmt.Printf("calculateOrderAndTx executeQuantity %v, takerAmount %v, makerAmount %v, executeAmount %v\n", new(big.Int).SetBytes(executeQuantity).String(), new(big.Int).SetBytes(takerAmount).String(), new(big.Int).SetBytes(makerAmount).String(), new(big.Int).SetBytes(executeAmount).String())
	takerFee, takerExecutedFee, takerOperatorFee, takerExecutedOperatorFee := CalculateFeeAndExecutedFee(taker, executeAmount, taker.TakerFeeRate, taker.TakerOperatorFeeRate, isDexFeeFork)
	makerFee, makerExecutedFee, makerOperatorFee, makerExecutedOperatorFee := CalculateFeeAndExecutedFee(maker, executeAmount, maker.MakerFeeRate, maker.MakerOperatorFeeRate, isDexFeeFork)
	updateOrder(taker, executeQuantity, executeAmount, takerExecutedFee, takerExecutedOperatorFee, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
	updateOrder(maker, executeQuantity, executeAmount, makerExecutedFee, makerExecutedOperatorFee, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
	tx.Quantity = executeQuantity
	tx.Amount = executeAmount
	tx.takerAddress = taker.Address
	tx.makerAddress = maker.Address
	tx.tradeToken = marketInfo.TradeToken
	tx.quoteToken = marketInfo.QuoteToken
	tx.TakerFee = takerFee
	tx.TakerOperatorFee = takerOperatorFee
	tx.MakerFee = makerFee
	tx.MakerOperatorFee = makerOperatorFee
	tx.Timestamp = taker.Timestamp
	return tx
}

func calculateOrderAmount(order *Order, quantity []byte, price []byte, decimalsDiff int32) []byte {
	amount := CalculateRawAmount(quantity, price, decimalsDiff)
	if !order.Side && new(big.Int).SetBytes(order.Amount).Cmp(new(big.Int).SetBytes(AddBigInt(order.ExecutedAmount, amount))) < 0 { // side is buy
		amount = SubBigIntAbs(order.Amount, order.ExecutedAmount)
	}
	return amount
}

func updateOrder(order *Order, quantity []byte, amount []byte, executedBaseFee, executedOperatorFee []byte, decimalsDiff int32) []byte {
	order.ExecutedAmount = AddBigInt(order.ExecutedAmount, amount)
	if bytes.Equal(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity) ||
		order.Type == Market && !order.Side && bytes.Equal(SubBigIntAbs(order.Amount, order.ExecutedAmount), amount) || // market buy order
		IsDust(order, quantity, decimalsDiff) {
		order.Status = FullyExecuted
	} else {
		order.Status = PartialExecuted
	}
	order.ExecutedBaseFee = executedBaseFee
	order.ExecutedOperatorFee = executedOperatorFee
	order.ExecutedQuantity = AddBigInt(order.ExecutedQuantity, quantity)
	return amount
}

// leave quantity is too small for calculate precision
func IsDust(order *Order, quantity []byte, decimalsDiff int32) bool {
	return CalculateRawAmountF(SubBigIntAbs(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity), order.Price, decimalsDiff).Cmp(new(big.Float).SetInt64(int64(1))) < 0
}

func CalculateRawAmount(quantity []byte, price []byte, decimalsDiff int32) []byte {
	return RoundAmount(CalculateRawAmountF(quantity, price, decimalsDiff)).Bytes()
}

func CalculateRawAmountF(quantity []byte, price []byte, decimalsDiff int32) *big.Float {
	qtF := new(big.Float).SetPrec(bigFloatPrec).SetInt(new(big.Int).SetBytes(quantity))
	prF, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(BytesToPrice(price))
	return AdjustForDecimalsDiff(new(big.Float).SetPrec(bigFloatPrec).Mul(prF, qtF), decimalsDiff)
}

func CalculateAmountForRate(amount []byte, rate int32) []byte {
	if rate > 0 {
		amtF := new(big.Float).SetPrec(bigFloatPrec).SetInt(new(big.Int).SetBytes(amount))
		rateF, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(CardinalRateToString(rate))
		return RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(amtF, rateF)).Bytes()
	} else {
		return nil
	}
}

func CalculateFeeAndExecutedFee(order *Order, amount []byte, feeRate, operatorFeeRate int32, isDexFeeFork bool) (incBaseFee, executedBaseFee, incOperatorFee, executedOperatorFee []byte) {
	var leaved bool
	if incBaseFee, executedBaseFee, leaved = calculateExecutedFee(amount, feeRate, order.Side, order.ExecutedBaseFee, order.LockedBuyFee, order.ExecutedBaseFee, order.ExecutedOperatorFee); leaved {
		incOperatorFee, executedOperatorFee, _ = calculateExecutedFee(amount, operatorFeeRate, order.Side, order.ExecutedOperatorFee, order.LockedBuyFee, executedBaseFee, order.ExecutedOperatorFee)
	} else if isDexFeeFork {
		executedOperatorFee = order.ExecutedOperatorFee
	}
	return
}

func calculateExecutedFee(amount []byte, feeRate int32, side bool, originExecutedFee, totalLockedAmount []byte, usedAmounts ...[]byte) (incFee, newExecutedFee []byte, leaved bool) {
	if feeRate == 0 {
		return nil, originExecutedFee, true
	}
	incFee = CalculateAmountForRate(amount, feeRate)
	switch side {
	case false:
		var totalUsedAmount []byte
		for _, usedAmt := range usedAmounts {
			totalUsedAmount = AddBigInt(totalUsedAmount, usedAmt)
		}
		if CmpForBigInt(totalLockedAmount, totalUsedAmount) <= 0 {
			incFee = nil
			newExecutedFee = originExecutedFee
		} else {
			totalUsedAmountNew := AddBigInt(totalUsedAmount, incFee)
			if CmpForBigInt(totalLockedAmount, totalUsedAmountNew) <= 0 {
				incFee = SubBigIntAbs(totalLockedAmount, totalUsedAmount)
			} else {
				leaved = true
			}
			newExecutedFee = AddBigInt(originExecutedFee, incFee)
		}
	case true:
		newExecutedFee = AddBigInt(originExecutedFee, incFee)
		leaved = true
	}
	return
}

func newLog(event DexEvent) *ledger.VmLog {
	log := &ledger.VmLog{}
	log.Topics = append(log.Topics, event.GetTopicId())
	log.Data = event.toDataBytes()
	return log
}

func matchPrice(taker, maker *Order) (matched bool, executedPrice []byte) {
	cmp := priceCompare(taker.Price, maker.Price)
	if taker.Type == Market || cmp == 0 {
		return true, maker.Price
	} else {
		matched = false
		switch taker.Side {
		case false: // buy
			matched = cmp >= 0
		case true: // sell
			matched = cmp <= 0
		}
		return matched, maker.Price
	}
}

func filterTimeout(db vm_db.VmDb, maker *Order) bool {
	if currentTime := GetTradeTimestamp(db); currentTime == 0 {
		return false
	} else if currentTime > maker.Timestamp+timeoutSecond {
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

func generateTxId(takerId []byte, makerId []byte) []byte {
	return crypto.Hash(txIdLength, takerId, makerId)
}
