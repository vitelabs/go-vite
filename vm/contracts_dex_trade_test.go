package vm

import (
	"bytes"
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
	dex.SetFeeRate(0.001, 0.001)
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
	_, err := method.DoSend(db, senderAccBlock, 100100100)
	assert.Equal(t, "invalid block source", err.Error())

	senderAccBlock.AccountAddress = types.AddressDexFund
	_, err = method.DoSend(db, senderAccBlock, 100100100)
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
	param := new(contracts.ParamDexSerializedData)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 4, len(actions.FundActions))
	for _, ac := range actions.FundActions {
		// sellOrder0
		if bytes.Equal([]byte(ac.Address), sellAddress0.Bytes()) {
			if bytes.Equal(ac.Token, ETH.tokenId.Bytes()) {
				assert.True(t, CheckBigEqualToInt(300, ac.ReduceLocked))
			} else if bytes.Equal(ac.Token, VITE.tokenId.Bytes()) {
				assert.True(t, CheckBigEqualToInt(929070, ac.IncAvailable)) // amount - feeExecuted
			}
		} else { // buyOrder1
			if bytes.Equal(ac.Token, ETH.tokenId.Bytes()) {
				assert.True(t, CheckBigEqualToInt(300, ac.IncAvailable))
			} else if bytes.Equal(ac.Token, VITE.tokenId.Bytes()) {
				assert.True(t, CheckBigEqualToInt(930930, ac.ReduceLocked)) // amount + feeExecuted
			}
		}
	}
}

func innerTestTradeCancelOrder(t *testing.T, db *testDatabase) {
	userAddress, _ := types.BytesToAddress([]byte("12345678901234567891"))
	userAddress1, _ := types.BytesToAddress([]byte("12345678901234567892"))
	userAddress2, _ := types.BytesToAddress([]byte("12345678901234567892"))

	method := contracts.MethodDexTradeCancelOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress1

	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(102), ETH.tokenId, VITE.tokenId, false)
	_, err := method.DoSend(db, senderAccBlock, 100100100)
	assert.Equal(t, "cancel order not own to initiator", err.Error())

	senderAccBlock.AccountAddress = userAddress2
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(202), ETH.tokenId, VITE.tokenId, true)
	_, err = method.DoSend(db, senderAccBlock, 100100100)
	assert.Equal(t, "order status is invalid to cancel", err.Error())

	// executedQuantity = 100,
	senderAccBlock.AccountAddress = userAddress
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(102), ETH.tokenId, VITE.tokenId, false)
	_, err = method.DoSend(db, senderAccBlock, 100100100)
	assert.Equal(t, nil, err)

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now
	receiveBlock.AccountAddress = types.AddressDexTrade

	var appendedBlocks []*contracts.SendBlock
	clearContext(db)
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.Equal(t, nil, err)

	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 1, len(appendedBlocks))
	assert.True(t, bytes.Equal(appendedBlocks[0].ToAddress.Bytes(), types.AddressDexFund.Bytes()))
	param := new(contracts.ParamDexSerializedData)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(actions.FundActions))
	assert.True(t, bytes.Equal(actions.FundActions[0].Token, VITE.tokenId.Bytes()))
	assert.True(t, CheckBigEqualToInt(350350, actions.FundActions[0].ReleaseLocked)) // 1281280 - 930930
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, orderIdBytesFromInt(102), ETH.tokenId, VITE.tokenId, false)
	_, err = method.DoSend(db, senderAccBlock, 100100100)
	assert.Equal(t, "order status is invalid to cancel", err.Error())
}

func initDexTradeDatabase()  *testDatabase {
	db := NewNoDatabase()
	db.addr = types.AddressDexTrade
	return db
}

func getNewOrderData(id int, address types.Address, tradeToken tokenInfo, quoteToken tokenInfo, side bool, price string, quantity int64) []byte {
	order := &dexproto.Order{}
	order.Id = orderIdBytesFromInt(id)
	order.Address = address.Bytes()
	order.TradeToken = tradeToken.tokenId.Bytes()
	order.QuoteToken = quoteToken.tokenId.Bytes()
	order.TradeTokenDecimals = tradeToken.decimals
	order.QuoteTokenDecimals = quoteToken.decimals
	order.Side = side //sell
	order.Type = dex.Limited
	order.Price = price
	order.Quantity = big.NewInt(quantity).Bytes()
	order.Status =  dex.Pending
	order.Amount = dex.CalculateRawAmount(order.Quantity, order.Price, order.TradeTokenDecimals, order.QuoteTokenDecimals)
	if order.Type == dex.Limited && !order.Side {//buy
		//fmt.Printf("newOrderInfo set LockedBuyFee id %v, order.Type %v, order.Side %v, order.Amount %v\n", id, order.Type, order.Side, order.Amount)
		order.LockedBuyFee = dex.CalculateRawFee(order.Amount, dex.MaxFeeRate())
	}
	order.Timestamp = time.Now().Unix()
	order.ExecutedQuantity = big.NewInt(0).Bytes()
	order.ExecutedAmount = big.NewInt(0).Bytes()
	order.RefundToken = []byte{}
	order.RefundQuantity = big.NewInt(0).Bytes()
	data, _ := proto.Marshal(order)
	return data
}

func clearContext(db *testDatabase) {
	db.logList = make([]*ledger.VmLog, 0)
}

