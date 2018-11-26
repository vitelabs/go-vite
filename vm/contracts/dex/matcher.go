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
	"strings"
	"time"
)

const maxTxsCountPerTaker = 1000
const timeoutMillisecond = 7 * 24 * 3600 * 1000

type matcher struct {
	contractAddress *types.Address
	storage         *BaseStorage
	protocol        *nodePayloadProtocol
	books           map[string]*skiplist
	settleActions   map[string]map[string]*proto.SettleAction // address->(token->settleAction)
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
	mc.books = make(map[string]*skiplist)
	mc.settleActions = make(map[string]map[string]*proto.SettleAction)
	return mc
}

func (mc *matcher) MatchOrder(taker Order) {
	var bookToTake *skiplist
	bookToTake = mc.getBookByName(getBookNameToTake(taker))
	if err := mc.doMatchTaker(taker, bookToTake); err != nil {
		fmt.Printf("Failed to match taker for %s", err.Error())
	}
}

func (mc *matcher) GetSettleActions() map[string]map[string]*proto.SettleAction {
	return mc.settleActions
}

func (mc *matcher) GetOrderByIdAndBookName(orderId uint64, makerBookName string) (*Order, error) {
	book := mc.getBookByName(makerBookName)
	//fmt.Printf("makerBookName %s, orderId %d\n", makerBookName, orderId)
	if pl, _, _, err := book.getByKey(orderKey{orderId}); err != nil {
		return nil, fmt.Errorf("failed get order by orderId")
	} else {
		od, _ := (*pl).(Order)
		return &od, nil
	}
}

func (mc *matcher) CancelOrderByIdAndBookName(order *Order, makerBookName string) (err error) {
	book := mc.getBookByName(makerBookName)
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
	if err = book.updatePayload(newOrderKey(order.Id), &pl); err != nil {
		return err
	}
	return book.delete(newOrderKey(order.Id))
}

func (mc *matcher) getBookByName(bookName string) *skiplist {
	if book, ok := mc.books[bookName]; !ok {
		book = newSkiplist(bookName, mc.contractAddress, mc.storage, mc.protocol)
		mc.books[bookName] = book
	}
	return mc.books[bookName]
}

func (mc *matcher) doMatchTaker(taker Order, makerBook *skiplist) error {
	if makerBook.length == 0 {
		mc.handleTakerRes(taker)
		return nil
	}
	modifiedMakers := make([]Order, 0, 20)
	txs := make([]OrderTx, 0, 20)
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
func (mc *matcher) recursiveTakeOrder(taker *Order, maker Order, makerBook *skiplist, modifiedMakers *[]Order, txs *[]OrderTx, nextOrderId nodeKeyType) error {
	if filterTimeout(&maker) {
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
			//fmt.Printf("failed update maker storage for err : %s\n", err.Error())
		}
		mc.emitOrderRes(maker)
	}
	size := len(makers)
	if size > 0 {
		if makers[size-1].Status == FullyExecuted || makers[size-1].Status == Cancelled {
			makerBook.truncateHeadTo(newOrderKey(makers[size-1].Id), size)
		} else if size >= 2 {
			makerBook.truncateHeadTo(newOrderKey(makers[size-2].Id), size-1)
		}
	}
}

func (mc *matcher) handleTakerRes(order Order) {
	if order.Status == PartialExecuted || order.Status == Pending {
		mc.saveTakerAsMaker(order)
	}
	mc.emitOrderRes(order)
}

func (mc *matcher) saveTakerAsMaker(maker Order) {
	bookToMake := mc.getBookByName(getBookNameToMakeForOrder(maker))
	pl := nodePayload(maker)
	bookToMake.insert(newOrderKey(maker.Id), &pl)
}

func (mc *matcher) emitOrderRes(orderRes Order) {
	event := OrderUpdateEvent{orderRes.Order}
	(*mc.storage).AddLog(newLog(event))
	//fmt.Printf("order matched res %s : \n", orderRes.String())
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
		actionMap map[string]*proto.SettleAction // token -> action
		ok bool
		ac *proto.SettleAction
		address = string(action.Address)
	)
	if actionMap, ok = mc.settleActions[address]; !ok {
		actionMap = make(map[string]*proto.SettleAction)
	}
	token := string(action.Token)
	if ac, ok = actionMap[string(token)]; !ok {
		ac = &proto.SettleAction{Address:[]byte(address), Token:action.Token}
	}
	ac.IncAvailable = AddBigInt(ac.IncAvailable, action.IncAvailable)
	ac.ReleaseLocked = AddBigInt(ac.ReleaseLocked, action.ReleaseLocked)
	ac.DeduceLocked = AddBigInt(ac.DeduceLocked, action.DeduceLocked)
	actionMap[string(token)] = ac
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

func filterTimeout(maker *Order) bool {
	if time.Now().Unix()*1000 > maker.Timestamp+timeoutMillisecond {
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
	tx.Id = getTxId(taker.Id, maker.Id)
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
	tx.Timestamp = time.Now().UnixNano() / 1000
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

func getMakerById(makerBook *skiplist, orderId nodeKeyType) (od Order, nextId orderKey, err error) {
	if pl, fwk, _, err := makerBook.getByKey(orderId); err != nil {
		return Order{}, (*makerBook.protocol).getNilKey().(orderKey), err
	} else {
		maker := (*pl).(Order)
		return maker, fwk.(orderKey), nil
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

func getBookNameToTake(order Order) string {
	return fmt.Sprintf("%s|%s|%d", string(order.TradeToken), string(order.QuoteToken), 1-toSideInt(order.Side))
}

func getBookNameToMakeForOrder(order Order) string {
	return GetBookNameToMake(order.TradeToken, order.QuoteToken, order.Side)
}

func GetBookNameToMake(tradeToken []byte, quoteToken []byte, side bool) string {
	return fmt.Sprintf("%s|%s|%d", string(tradeToken), string(quoteToken), toSideInt(side))
}

func toSideInt(side bool) int {
	sideInt := 0 // buy
	if side {
		sideInt = 1 // sell
	}
	return sideInt
}

func getTxId(takerId uint64, makerId uint64) uint64 {
	data := crypto.Hash(8, []byte(fmt.Sprintf("%d-%d", takerId, makerId)))
	var num uint64
	binary.Read(bytes.NewBuffer(data[:]), binary.LittleEndian, &num)
	return num
}
