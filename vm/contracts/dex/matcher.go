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

const maxTxsCountPerTaker = 1000
const timeoutSecond = 7 * 24 * 3600
const txIdLength = 20
const bigFloatPrec = 120

type Matcher struct {
	db          vm_db.VmDb
	MarketInfo  *MarketInfo
	fundSettles map[types.Address]map[types.TokenTypeId]*proto.FundSettle
	feeSettles  map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle
}

type OrderTx struct {
	proto.Transaction
	takerAddress []byte
	makerAddress []byte
	tradeToken   []byte
	quoteToken   []byte
}

var (
	TakerFeeRate = "0.0025"
	MakerFeeRate = "0.0025"
)

func NewMatcher(db vm_db.VmDb, marketId int32) (mc *Matcher, err error) {
	mc = NewRawMatcher(db)
	var ok bool
	if mc.MarketInfo, ok = GetMarketInfoById(db, marketId); !ok {
		return nil, TradeMarketNotExistsError
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
	mc.fundSettles = make(map[types.Address]map[types.TokenTypeId]*proto.FundSettle)
	mc.feeSettles = make(map[types.TokenTypeId]map[types.Address]*proto.UserFeeSettle)
	return
}

func (mc *Matcher) MatchOrder(taker *Order) (err error) {
	var bookToTake *levelDbBook
	if bookToTake, err = mc.getMakerBookToTaker(taker.Side); err != nil {
		return err
	} else {
		defer bookToTake.release()
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
	mc.deleteOrder(order.Id)
}

func (mc *Matcher) doMatchTaker(taker *Order, makerBook *levelDbBook) (err error) {
	modifiedMakers := make([]*Order, 0, 20)
	txs := make([]*OrderTx, 0, 20)
	if maker, ok := makerBook.nextOrder(); !ok {
		mc.handleTakerRes(taker)
		return
	} else {
		// must not set db in recursiveTakeOrder
		if err = mc.recursiveTakeOrder(taker, maker, makerBook, &modifiedMakers, &txs); err != nil {
			return
		} else {
			mc.handleTakerRes(taker)
			mc.handleModifiedMakers(modifiedMakers)
			mc.handleTxs(txs)
		}
	}
	return
}

//TODO add assertion for order calculation correctness
func (mc *Matcher) recursiveTakeOrder(taker, maker *Order, makerBook *levelDbBook, modifiedMakers *[]*Order, txs *[]*OrderTx) error {
	if filterTimeout(taker.Timestamp, maker) {
		*modifiedMakers = append(*modifiedMakers, maker)
	} else {
		matched, _ := matchPrice(taker, maker)
		//fmt.Printf("recursiveTakeOrder matched for taker.id %d is %t\n", taker.Id, matched)
		if matched {
			tx := calculateOrderAndTx(taker, maker, mc.MarketInfo)
			*txs = append(*txs, tx)
			if taker.Status == PartialExecuted && len(*txs) >= maxTxsCountPerTaker {
				taker.Status = Cancelled
				taker.CancelReason = partialExecutedCancelledByMarket
			}
			*modifiedMakers = append(*modifiedMakers, maker)
		}
	}
	if taker.Status == FullyExecuted || taker.Status == Cancelled {
		return nil
	}
	if newMaker, ok := makerBook.nextOrder(); ok {
		return mc.recursiveTakeOrder(taker, newMaker, makerBook, modifiedMakers, txs)
	} else {
		return nil
	}
}

func (mc *Matcher) handleTakerRes(taker *Order) {
	if taker.Status == PartialExecuted || taker.Status == Pending {
		mc.saveOrder(*taker)
	} else {// in case isDust still need refund FullExecuted status order
		mc.handleRefund(taker)
	}
	mc.emitNewOrder(*taker)
}

func (mc *Matcher) handleModifiedMakers(makers []*Order) {
	for _, maker := range makers {
		if maker.Status == FullyExecuted || maker.Status == Cancelled {
			mc.handleRefund(maker)// in case isDust still need refund FullExecuted status order
			mc.deleteOrder(maker.Id)
		} else {
			mc.saveOrder(*maker)
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
			refundFee := SubBigIntAbs(order.LockedBuyFee, order.ExecutedFee)
			order.RefundQuantity = AddBigInt(refundAmount, refundFee)
		case true:
			order.RefundToken = mc.MarketInfo.TradeToken
			order.RefundQuantity = SubBigIntAbs(order.Quantity, order.ExecutedQuantity)
		}
		if CmpToBigZero(order.RefundQuantity) > 0 {
			mc.updateFundSettle(order.Address, proto.FundSettle{Token: order.RefundToken, ReleaseLocked: order.RefundQuantity})
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
	updateInfo.ExecutedFee = order.ExecutedFee
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

func (mc *Matcher) getMakerBookToTaker(takerSide bool) (*levelDbBook, error) {
	return getMakerBook(mc.db, mc.MarketInfo.MarketId, !takerSide)
}

func (mc *Matcher) saveOrder(order Order) {
	orderId := order.Id
	if data, err := order.SerializeCompact(); err != nil {
		panic(err)
	} else {
		setValueToDb(mc.db, orderId, data)
	}
}

func (mc *Matcher) deleteOrder(orderId []byte) {
	setValueToDb(mc.db, orderId, nil)
}

func calculateOrderAndTx(taker, maker *Order, marketInfo *MarketInfo) (tx *OrderTx) {
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
	takerFee, takerExecutedFee := CalculateFeeAndExecutedFee(taker, executeAmount, TakerFeeRate)
	makerFee, makerExecutedFee := CalculateFeeAndExecutedFee(maker, executeAmount, MakerFeeRate)
	updateOrder(taker, executeQuantity, executeAmount, takerExecutedFee, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
	updateOrder(maker, executeQuantity, executeAmount, makerExecutedFee, marketInfo.TradeTokenDecimals-marketInfo.QuoteTokenDecimals)
	tx.Quantity = executeQuantity
	tx.Amount = executeAmount
	tx.takerAddress = taker.Address
	tx.makerAddress = maker.Address
	tx.tradeToken = marketInfo.TradeToken
	tx.quoteToken = marketInfo.QuoteToken
	tx.makerAddress = maker.Address
	tx.TakerFee = takerFee
	tx.MakerFee = makerFee
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

func updateOrder(order *Order, quantity []byte, amount []byte, executedFee []byte, decimalsDiff int32) []byte {
	order.ExecutedAmount = AddBigInt(order.ExecutedAmount, amount)
	if bytes.Equal(SubBigIntAbs(order.Quantity, order.ExecutedQuantity), quantity) ||
		order.Type == Market && !order.Side && bytes.Equal(SubBigIntAbs(order.Amount, order.ExecutedAmount), amount) || // market buy order
		IsDust(order, quantity, decimalsDiff) {
		order.Status = FullyExecuted
	} else {
		order.Status = PartialExecuted
	}
	order.ExecutedFee = executedFee
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

func CalculateRawFee(amount []byte, feeRate string) []byte {
	amtF := new(big.Float).SetPrec(bigFloatPrec).SetInt(new(big.Int).SetBytes(amount))
	rateF, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(feeRate)
	amtFee := amtF.Mul(amtF, rateF)
	return RoundAmount(amtFee).Bytes()
}

func CalculateFeeAndExecutedFee(order *Order, amount []byte, feeRate string) (feeBytes, executedFee []byte) {
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

func newLog(event OrderEvent) *ledger.VmLog {
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

//TODO support timeout when timer trigger available for set raw timestamp in one hour for order timestamp
func filterTimeout(takerTimestamp int64, maker *Order) bool {
	return false
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
