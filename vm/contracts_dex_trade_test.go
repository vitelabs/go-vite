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
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"testing"
	"time"
)

func TestDexTrade(t *testing.T) {
	db := initDexTradeDatabase()
	innerTestTradeNewOrder(t, db)
	innerTestTradeCancelOrder(t, db)
}

func innerTestTradeNewOrder(t *testing.T, db *testDatabase) {
	buyAddress0, _ := types.BytesToAddress([]byte("12345678901234567890"))
	buyAddress1, _ := types.BytesToAddress([]byte("12345678901234567891"))

	sellAddress0, _ := types.BytesToAddress([]byte("12345678901234567892"))

	method := contracts.MethodDexTradeNewOrder{}
	senderVmBlock := &vm_context.VmAccountBlock{}
	senderVmBlock.VmContext = db

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = buyAddress0
	senderVmBlock.AccountBlock = senderAccBlock
	buyOrder0 := getNewOrderData(101, buyAddress0, ETH, VITE, false, 30, 10)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, buyOrder0)
	_, err := method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	assert.Equal(t, "invalid block source", err.Error())

	senderAccBlock.AccountAddress = contracts.AddressDexFund
	_, err = method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	assert.True(t,  err == nil)

	receiveVmBlock := &vm_context.VmAccountBlock{}
	receiveVmBlock.VmContext = db
	receiveVmBlock.AccountBlock = &ledger.AccountBlock{}
	now := time.Now()
	receiveVmBlock.AccountBlock.Timestamp = &now
	receiveVmBlock.AccountBlock.AccountAddress = contracts.AddressDexTrade

	context := &vmMockVmCtx{}
	err = method.DoReceive(context, receiveVmBlock, senderAccBlock)
	assert.True(t, err == nil)
	assert.True(t, len(db.logList) == 1)
	assert.Equal(t, 0, len(context.appendedBlocks))

	clearContext(context, db)
	sellOrder0 := getNewOrderData(202, sellAddress0, ETH, VITE, true, 31, 300)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, sellOrder0)
	err = method.DoReceive(context, receiveVmBlock, senderAccBlock)
	assert.True(t, err == nil)
	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 0, len(context.appendedBlocks))

	clearContext(context, db)
	buyOrder1 := getNewOrderData(102, buyAddress1, ETH, VITE, false, 32, 400)
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeNewOrder, buyOrder1)
	err = method.DoReceive(context, receiveVmBlock, senderAccBlock)
	assert.Equal(t, 3, len(db.logList))
	assert.Equal(t, 1, len(context.appendedBlocks))
	assert.True(t, bytes.Equal(context.appendedBlocks[0].AccountBlock.AccountAddress.Bytes(), contracts.AddressDexTrade.Bytes()))
	assert.True(t, bytes.Equal(context.appendedBlocks[0].AccountBlock.ToAddress.Bytes(), contracts.AddressDexFund.Bytes()))
	param := new(contracts.ParamDexSerializedData)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, context.appendedBlocks[0].AccountBlock.Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 4, len(actions.Actions))
	for _, ac := range actions.Actions {
		// sellOrder0
		if bytes.Equal([]byte(ac.Address), sellAddress0.Bytes()) {
			if bytes.Equal(ac.Token, ETH.Bytes()) {
				assert.Equal(t, ac.DeduceLocked, uint64(300))
			} else if bytes.Equal(ac.Token, VITE.Bytes()) {
				assert.Equal(t, ac.IncAvailable, uint64(9300))
			}
		} else {
			if bytes.Equal(ac.Token, ETH.Bytes()) {
				assert.Equal(t, ac.IncAvailable, uint64(300))
			} else if bytes.Equal(ac.Token, VITE.Bytes()) {
				assert.Equal(t, ac.DeduceLocked, uint64(9300))
			}
		}
	}
}

func innerTestTradeCancelOrder(t *testing.T, db *testDatabase) {
	userAddress, _ := types.BytesToAddress([]byte("12345678901234567891"))
	userAddress1, _ := types.BytesToAddress([]byte("12345678901234567892"))
	userAddress2, _ := types.BytesToAddress([]byte("12345678901234567892"))

	method := contracts.MethodDexTradeCancelOrder{}
	senderVmBlock := &vm_context.VmAccountBlock{}
	senderVmBlock.VmContext = db

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress1
	senderVmBlock.AccountBlock = senderAccBlock

	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, big.NewInt(102), ETH, VITE, false)
	_, err := method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	assert.Equal(t, "cancel order not own to initiator", err.Error())

	senderAccBlock.AccountAddress = userAddress2
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, big.NewInt(202), ETH, VITE, true)
	_, err = method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	assert.Equal(t, "order status is invalid to cancel", err.Error())

	senderAccBlock.AccountAddress = userAddress
	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, big.NewInt(102), ETH, VITE, false)
	_, err = method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	assert.Equal(t, nil, err)

	receiveVmBlock := &vm_context.VmAccountBlock{}
	receiveVmBlock.VmContext = db
	receiveVmBlock.AccountBlock = &ledger.AccountBlock{}
	now := time.Now()
	receiveVmBlock.AccountBlock.Timestamp = &now
	receiveVmBlock.AccountBlock.AccountAddress = contracts.AddressDexTrade

	context := &vmMockVmCtx{}
	clearContext(context, db)
	err = method.DoReceive(context, receiveVmBlock, senderAccBlock)
	assert.Equal(t, nil, err)

	assert.Equal(t, 1, len(db.logList))
	assert.Equal(t, 1, len(context.appendedBlocks))
	assert.True(t, bytes.Equal(context.appendedBlocks[0].AccountBlock.ToAddress.Bytes(), contracts.AddressDexFund.Bytes()))
	param := new(contracts.ParamDexSerializedData)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundSettleOrders, context.appendedBlocks[0].AccountBlock.Data)
	assert.Equal(t, nil, err)
	actions := &dexproto.SettleActions{}
	err = proto.Unmarshal(param.Data, actions)
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(actions.Actions))
	assert.True(t, bytes.Equal(actions.Actions[0].Token, VITE.Bytes()))
	assert.Equal(t, uint64(3500), actions.Actions[0].ReleaseLocked)

	senderAccBlock.Data, _ = contracts.ABIDexTrade.PackMethod(contracts.MethodNameDexTradeCancelOrder, big.NewInt(102), ETH, VITE, false)
	_, err = method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	assert.Equal(t, "order status is invalid to cancel", err.Error())
}

func initDexTradeDatabase()  *testDatabase {
	db := NewNoDatabase()
	db.addr = contracts.AddressDexTrade
	return db
}

func getNewOrderData(id uint64, address types.Address, tradeToken types.TokenTypeId, quoteToken types.TokenTypeId, side bool, price float64, quantity uint64) []byte {
	order := &dexproto.Order{}
	order.Id = id
	order.Address = string(address.Bytes())
	order.TradeToken = tradeToken.Bytes()
	order.QuoteToken = quoteToken.Bytes()
	order.Side = side //sell
	order.Type = dex.Limited
	order.Price = price
	order.Quantity = quantity
	order.Amount = dex.CalculateAmount(order.Quantity, order.Price)
	order.Status =  dex.FullyExecuted
	order.Timestamp = time.Now().UnixNano()/1000
	order.ExecutedQuantity = 0
	order.ExecutedAmount = 0
	order.RefundToken = []byte{}
	order.RefundQuantity = 0
	data, _ := proto.Marshal(order)
	return data
}

func clearContext(ctx *vmMockVmCtx, db *testDatabase) {
	ctx.appendedBlocks = make([]*vm_context.VmAccountBlock, 0)
	db.logList = make([]*ledger.VmLog, 0)
}

