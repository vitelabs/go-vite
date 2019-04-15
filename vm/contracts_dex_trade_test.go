package vm

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"testing"
	"time"
)

func TestDexTrade(t *testing.T) {
	db := initDexTradeDatabase()
	dex.SetFeeRate("0.001", "0.001")
	innerTestTradeNewOrder(t, db)
	innerTestTradeCancelOrder(t, db)
}

func innerTestTradeNewOrder(t *testing.T, db *testDatabase) {
	buyAddress0, _ := types.BytesToAddress([]byte("12345678901234567890"))
	buyAddress1, _ := types.BytesToAddress([]byte("12345678901234567891"))

	sellAddress0, _ := types.BytesToAddress([]byte("12345678901234567892"))

	method := contracts.MethodDexTradeNewOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = buyAddress0
	buyOrder0 := getNewOrderData(101, buyAddress0, ETH, VITE, false, "30", 10)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, buyOrder0)
	err := method.DoSend(db, senderAccBlock)
	assert.Equal(t, "invalid block source", err.Error())

	senderAccBlock.AccountAddress = types.AddressDexFund
	err = method.DoSend(db, senderAccBlock)
	assert.True(t,  err == nil)

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now
	receiveBlock.AccountAddress = types.AddressDexTrade
	var appendedBlocks []*contracts.SendBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	assert.True(t, len(db.logList) == 1)
	assert.Equal(t, 0, len(appendedBlocks))

	clearContext(db)
	sellOrder0 := getNewOrderData(202, sellAddress0, ETH, VITE, true, "31", 300)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, sellOrder0)
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 0, len(appendedBlocks))

	clearContext(db)
	// locked = 400 * 32 * 1.001 * 100 = 1281280
	buyOrder1 := getNewOrderData(102, buyAddress1, ETH, VITE, false, "32", 400)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, buyOrder1)
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.Equal(t, 3, len(db.logList))
	assert.Equal(t, 1, len(appendedBlocks))
	assert.True(t, bytes.Equal(appendedBlocks[0].Block.AccountAddress.Bytes(), types.AddressDexTrade.Bytes()))
	assert.True(t, bytes.Equal(appendedBlocks[0].ToAddress.Bytes(), types.AddressDexFund.Bytes()))
	param := new(dex.ParamDexSerializedData)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(actions.FundActions))
	for _, ac := range actions.FundActions {
		// sellOrder0
		if bytes.Equal([]byte(ac.Address), sellAddress0.Bytes()) {
			for _, fundSettle := range ac.FundSettles {
				if bytes.Equal(fundSettle.Token, ETH.tokenId.Bytes()) {
					assert.True(t, CheckBigEqualToInt(300, fundSettle.ReduceLocked))
				} else if bytes.Equal(fundSettle.Token, VITE.tokenId.Bytes()) {
					assert.True(t, CheckBigEqualToInt(929070, fundSettle.IncAvailable)) // amount - feeExecuted
				}
			}
		} else { // buyOrder1
			for _, fundSettle := range ac.FundSettles {
				if bytes.Equal(fundSettle.Token, ETH.tokenId.Bytes()) {
					assert.True(t, CheckBigEqualToInt(300, fundSettle.IncAvailable))
				} else if bytes.Equal(fundSettle.Token, VITE.tokenId.Bytes()) {
					assert.True(t, CheckBigEqualToInt(930930, fundSettle.ReduceLocked)) // amount + feeExecuted
				}
			}
		}
	}

	duplicateOrder := getNewOrderData(102, buyAddress0, ETH, VITE, false, "100", 10)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, duplicateOrder)
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(appendedBlocks))
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(actions.FundActions))
	userFundSettle := actions.FundActions[0]
	assert.Equal(t, buyAddress0.Bytes(), userFundSettle.Address)
	assert.Equal(t, 1, len(userFundSettle.FundSettles))
	fundSettle := userFundSettle.FundSettles[0]
	assert.Equal(t, VITE.tokenId.Bytes(), fundSettle.Token)
	assert.True(t, CheckBigEqualToInt(100100, fundSettle.ReleaseLocked))
}

func innerTestTradeCancelOrder(t *testing.T, db *testDatabase) {
	userAddress, _ := types.BytesToAddress([]byte("12345678901234567891"))
	userAddress1, _ := types.BytesToAddress([]byte("12345678901234567892"))

	method := contracts.MethodDexTradeCancelOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress1

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now
	receiveBlock.AccountAddress = types.AddressDexTrade

	clearContext(db)

	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(102), ETH.tokenId, VITE.tokenId, false)
	_, err := method.DoReceive(db, receiveBlock, senderAccBlock)
	invalidOwnerEvent := dex.ErrEvent{}
	assert.Equal(t, invalidOwnerEvent.GetTopicId().Bytes(), db.logList[0].Topics[0].Bytes())
	invalidOwnerEvent, _ = invalidOwnerEvent.FromBytes(db.logList[0].Data).(dex.ErrEvent)
	assert.Equal(t, dex.CancelOrderOwnerInvalidErr.Error(), invalidOwnerEvent.Error())

	clearContext(db)

	senderAccBlock.AccountAddress = userAddress2
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(202), ETH.tokenId, VITE.tokenId, true)
	method.DoReceive(db, receiveBlock, senderAccBlock)
	getOrderFailEvent := dex.ErrEvent{}
	assert.Equal(t, getOrderFailEvent.GetTopicId().Bytes(), db.logList[0].Topics[0].Bytes())
	getOrderFailEvent, _ = getOrderFailEvent.FromBytes(db.logList[0].Data).(dex.ErrEvent)
	assert.Equal(t, dex.GetOrderByIdFailedErr.Error(), getOrderFailEvent.Error())

	clearContext(db)

	// executedQuantity = 100,
	senderAccBlock.AccountAddress = userAddress
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(102), ETH.tokenId, VITE.tokenId, false)

	var appendedBlocks []*contracts.SendBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.Equal(t, nil, err)

	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 1, len(appendedBlocks))
	assert.True(t, bytes.Equal(appendedBlocks[0].ToAddress.Bytes(), types.AddressDexFund.Bytes()))
	param := new(dex.ParamDexSerializedData)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(actions.FundActions))
	assert.True(t, bytes.Equal(actions.FundActions[0].FundSettles[0].Token, VITE.tokenId.Bytes()))
	assert.True(t, CheckBigEqualToInt(350350, actions.FundActions[0].FundSettles[0].ReleaseLocked)) // 1281280 - 930930

	clearContext(db)

	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(102), ETH.tokenId, VITE.tokenId, false)
	method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.Equal(t, getOrderFailEvent.GetTopicId().Bytes(), db.logList[0].Topics[0].Bytes())
	getOrderFailEvent, _ = getOrderFailEvent.FromBytes(db.logList[0].Data).(dex.ErrEvent)
	assert.Equal(t, dex.GetOrderByIdFailedErr.Error(), getOrderFailEvent.Error())
}

func initDexTradeDatabase()  *testDatabase {
	db := NewNoDatabase()
	db.addr = types.AddressDexTrade
	return db
}

func getNewOrderData(id int, address types.Address, tradeToken tokenInfo, quoteToken tokenInfo, side bool, price string, quantity int64) []byte {
	tokenInfo := &dexproto.OrderTokenInfo{}
	tokenInfo.TradeToken = tradeToken.tokenId.Bytes()
	tokenInfo.QuoteToken = quoteToken.tokenId.Bytes()
	tokenInfo.TradeTokenDecimals = tradeToken.decimals
	tokenInfo.QuoteTokenDecimals = quoteToken.decimals
	fmt.Printf("")
	order := &dexproto.Order{}
	order.Id = orderIdBytesFromInt(id)
	order.Address = address.Bytes()
	order.Side = side
	order.Type = dex.Limited
	order.Price = price
	order.Quantity = big.NewInt(quantity).Bytes()
	order.Status =  dex.Pending
	order.Amount = dex.CalculateRawAmount(order.Quantity, order.Price, tokenInfo.TradeTokenDecimals, tokenInfo.QuoteTokenDecimals)
	if order.Type == dex.Limited && !order.Side {//buy
		//fmt.Printf("newOrderInfo set LockedBuyFee id %v, order.Type %v, order.Side %v, order.Amount %v\n", id, order.Type, order.Side, order.Amount)
		order.LockedBuyFee = dex.CalculateRawFee(order.Amount, dex.MaxFeeRate())
	}
	//order.Timestamp = time.Now().Unix()
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
	orderInfo := &dexproto.OrderInfo{}
	orderInfo.OrderTokenInfo = tokenInfo
	orderInfo.Order = order
	data, _ := proto.Marshal(orderInfo)
	return data
}

func clearContext(db *testDatabase) {
	db.logList = make([]*ledger.VmLog, 0)
}

