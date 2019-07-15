package vm

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"testing"
)

func TestDexTrade(t *testing.T) {
	initDexTradeDbAndMarket()
	dex.SetFeeRate("0.001", "0.001")
	innerTestTradeNewOrder(t)
	innerTestTradeCancelOrder(t)
	innerTestOnNewOrderFailed(t)
}

var buyOrder0, sellOrder0, buyOrder1 []byte
var sellOrder0Id, buyOrder1Id []byte

func innerTestTradeNewOrder(t *testing.T) {
	buyAddress0, _ := types.BytesToAddress([]byte("123456789012345678901"))
	buyAddress1, _ := types.BytesToAddress([]byte("123456789012345678902"))

	sellAddress0, _ := types.BytesToAddress([]byte("123456789012345678903"))

	method := contracts.MethodDexTradeNewOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = buyAddress0
	buyOrder0, _ = getNewOrderData(buyAddress0, ETH, VITE, false, "30", 10) //101
	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeNewOrder, buyOrder0)
	err := method.DoSend(db, senderAccBlock)
	assert.Equal(t, dex.InvalidSourceAddressErr, err)

	senderAccBlock.AccountAddress = types.AddressDexFund
	err = method.DoSend(db, senderAccBlock)
	assert.True(t, err == nil)

	receiveBlock := &ledger.AccountBlock{}
	receiveBlock.AccountAddress = types.AddressDexTrade
	var appendedBlocks []*ledger.AccountBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.True(t, err == nil)
	assert.True(t, len(db.logList) == 1)
	assert.Equal(t, 0, len(appendedBlocks))

	clearLogs(db)
	sellOrder0, sellOrder0Id = getNewOrderData(sellAddress0, ETH, VITE, true, "31", 300) //201
	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeNewOrder, sellOrder0)
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.True(t, err == nil)
	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 0, len(appendedBlocks))

	clearLogs(db)
	// locked = 400 * 32 * 1.001 * 100 = 1281280
	buyOrder1, buyOrder1Id = getNewOrderData(buyAddress1, ETH, VITE, false, "32", 400) //102
	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeNewOrder, buyOrder1)
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, 3, len(db.logList))
	assert.Equal(t, 1, len(appendedBlocks))
	assert.True(t, bytes.Equal(appendedBlocks[0].AccountAddress.Bytes(), types.AddressDexTrade.Bytes()))
	assert.True(t, bytes.Equal(appendedBlocks[0].ToAddress.Bytes(), types.AddressDexFund.Bytes()))
	param := new(dex.ParamDexSerializedData)
	err = abi.ABIDexFund.UnpackMethod(param, abi.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
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
					assert.True(t, checkBigEqualToInt(300, fundSettle.ReduceLocked))
				} else if bytes.Equal(fundSettle.Token, VITE.tokenId.Bytes()) {
					assert.True(t, checkBigEqualToInt(929070, fundSettle.IncAvailable)) // amount - feeExecuted
				}
			}
		} else { // buyOrder1
			for _, fundSettle := range ac.FundSettles {
				if bytes.Equal(fundSettle.Token, ETH.tokenId.Bytes()) {
					assert.True(t, checkBigEqualToInt(300, fundSettle.IncAvailable))
				} else if bytes.Equal(fundSettle.Token, VITE.tokenId.Bytes()) {
					assert.True(t, checkBigEqualToInt(930930, fundSettle.ReduceLocked)) // amount + feeExecuted
				}
			}
		}
	}
}

func innerTestOnNewOrderFailed(t *testing.T) {
	buyAddress0, _ := types.BytesToAddress([]byte("123456789012345678901"))
	order, _ := newOrderInfo(ETH.tokenId, VITE.tokenId, false, dex.Limited, "100", 10)
	order.Address = buyAddress0.Bytes()
	marketInfo, _ := dex.GetMarketInfoById(db, 1)
	appendedBlocks, err := contracts.OnNewOrderFailed(order, marketInfo)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(appendedBlocks))
	param := new(dex.ParamDexSerializedData)
	err = abi.ABIDexFund.UnpackMethod(param, abi.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(actions.FundActions))
	userFundSettle := actions.FundActions[0]
	assert.Equal(t, buyAddress0.Bytes(), userFundSettle.Address)
	assert.Equal(t, 1, len(userFundSettle.FundSettles))
	fundSettle := userFundSettle.FundSettles[0]
	assert.Equal(t, VITE.tokenId.Bytes(), fundSettle.Token)
	assert.True(t, checkBigEqualToInt(100100, fundSettle.ReleaseLocked))
}

func innerTestTradeCancelOrder(t *testing.T) {
	userAddress, _ := types.BytesToAddress([]byte("123456789012345678902"))
	userAddress1, _ := types.BytesToAddress([]byte("123456789012345678903"))

	method := contracts.MethodDexTradeCancelOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress1

	receiveBlock := &ledger.AccountBlock{}
	receiveBlock.AccountAddress = types.AddressDexTrade

	clearLogs(db)

	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrder, buyOrder1Id)
	_, err := method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, dex.CancelOrderOwnerInvalidErr, err)

	clearLogs(db)

	senderAccBlock.AccountAddress = userAddress2
	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrder, sellOrder0Id)
	_, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, dex.OrderNotExistsErr, err)

	clearLogs(db)

	// executedQuantity = 100,
	senderAccBlock.AccountAddress = userAddress
	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrder, buyOrder1Id)

	var appendedBlocks []*ledger.AccountBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, nil, err)

	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 1, len(appendedBlocks))
	assert.True(t, bytes.Equal(appendedBlocks[0].ToAddress.Bytes(), types.AddressDexFund.Bytes()))
	param := new(dex.ParamDexSerializedData)
	err = abi.ABIDexFund.UnpackMethod(param, abi.MethodNameDexFundSettleOrders, appendedBlocks[0].Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(actions.FundActions))
	assert.True(t, bytes.Equal(actions.FundActions[0].FundSettles[0].Token, VITE.tokenId.Bytes()))
	assert.True(t, checkBigEqualToInt(350350, actions.FundActions[0].FundSettles[0].ReleaseLocked)) // 1281280 - 930930

	clearLogs(db)

	senderAccBlock.Data, _ = abi.ABIDexTrade.PackMethod(abi.MethodNameDexTradeCancelOrder, buyOrder1Id)
	_, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, err, dex.OrderNotExistsErr)
}

func initDexTradeDbAndMarket() int32 {
	var marketId int32 = 1
	db = NewNoDatabase()
	db.addr = types.AddressDexTrade
	initTimerTimestamp()
	dex.SaveMarketInfo(db, getMarketInfo(marketId, ETH, VITE), ETH.tokenId, VITE.tokenId)
	dex.SaveMarketInfoById(db, getMarketInfo(marketId, ETH, VITE))
	return marketId
}

func getNewOrderData(address types.Address, tradeToken tokenInfo, quoteToken tokenInfo, side bool, price string, quantity uint64) ([]byte, []byte) {
	order, orderId := newOrderInfo(tradeToken.tokenId, quoteToken.tokenId, side, dex.Limited, price, quantity)
	order.Address = address.Bytes()
	data, _ := order.Serialize()
	return data, orderId
}

func clearLogs(db *testDatabase) {
	db.logList = make([]*ledger.VmLog, 0)
}
