package vm

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"testing"
	"time"
)

var (
	VITE = types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}
	ETH  = types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}
	NANO = types.TokenTypeId{'N', 'A', 'N', 'O', ' ', 'T', 'O', 'K', 'E', 'N'}
)

type vmMockVmCtx struct {
	appendedBlocks []*vm_context.VmAccountBlock
}

func (ctx *vmMockVmCtx) AppendBlock(block *vm_context.VmAccountBlock) {
	ctx.appendedBlocks = append(ctx.appendedBlocks, block)
}

func (ctx *vmMockVmCtx) GetNewBlockHeight(block *vm_context.VmAccountBlock) uint64 {
	return 100
}

func TestDexFund(t *testing.T) {
	db := initDexFundDatabase()
	userAddress, _ := types.BytesToAddress([]byte("12345678901234567890"))
	depositCash(db, userAddress, 3000, VITE)
	innerTestDepositAndWithdraw(t, db, userAddress)
	innerTestFundNewOrder(t, db, userAddress)
	innerTestSettleOrder(t, db, userAddress)
}

func innerTestDepositAndWithdraw(t *testing.T, db *testDatabase, userAddress types.Address) {
	var err error
	registerToken(db, VITE)
	//deposit
	depositMethod := contracts.MethodDexFundUserDeposit{}

	depositSendAccBlock := &ledger.AccountBlock{}
	depositSendAccBlock.AccountAddress = userAddress

	depositSendAccBlock.TokenId = ETH
	depositSendAccBlock.Amount = big.NewInt(100)

	depositSendAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	_, err = depositMethod.DoSend(db, depositSendAccBlock, 100010001000)
	assert.True(t, err != nil)
	assert.True(t, bytes.Equal([]byte(err.Error()), []byte("token is invalid")))

	depositSendAccBlock.TokenId = VITE
	depositSendAccBlock.Amount = big.NewInt(3000)

	depositSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	_, err = depositMethod.DoSend(db, depositSendAccBlock, 100010001000)
	assert.True(t, err == nil)
	assert.True(t, bytes.Equal(depositSendAccBlock.TokenId.Bytes(), VITE.Bytes()))
	assert.Equal(t, depositSendAccBlock.Amount.Uint64(), uint64(3000))

	depositReceiveBlock := &ledger.AccountBlock{}

	_, err = depositMethod.DoReceive(db, depositReceiveBlock, depositSendAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := contracts.GetFundFromStorage(db, userAddress)
	assert.Equal(t, 1, len(dexFund.Accounts))
	acc := dexFund.Accounts[0]
	assert.True(t, bytes.Equal(acc.Token, VITE.Bytes()))
	assert.True(t, CheckBigEqualToInt(3000, acc.Available))
	assert.True(t, CheckBigEqualToInt(0, acc.Locked))

	//withdraw
	withdrawMethod := contracts.MethodDexFundUserWithdraw{}

	withdrawSendAccBlock := &ledger.AccountBlock{}
	withdrawSendAccBlock.AccountAddress = userAddress
	withdrawSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, userAddress, VITE, big.NewInt(200))
	_, err = withdrawMethod.DoSend(db, withdrawSendAccBlock, 100010001000)
	assert.True(t, err == nil)

	withdrawReceiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	withdrawReceiveBlock.Timestamp = &now

	withdrawSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, userAddress, VITE, big.NewInt(200))
	appendedBlocks, _ := withdrawMethod.DoReceive(db, withdrawReceiveBlock, withdrawSendAccBlock)

	dexFund, _ = contracts.GetFundFromStorage(db, userAddress)
	acc = dexFund.Accounts[0]
	assert.True(t, CheckBigEqualToInt(2800, acc.Available))
	assert.Equal(t, 1, len(appendedBlocks))

}

func innerTestFundNewOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	registerToken(db, ETH)

	method := contracts.MethodDexFundNewOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress
	order := dexproto.Order{}
	order.Id = uint64(1)
	order.Address = userAddress.Bytes()
	order.TradeToken = VITE.Bytes()
	order.QuoteToken = ETH.Bytes()
	order.Side = true //sell
	order.Type = dex.Limited
	order.Price = "0.03"
	order.Quantity = big.NewInt(2000).Bytes()
	order.Amount = big.NewInt(0).Bytes()
	order.Status = dex.FullyExecuted
	order.Timestamp = time.Now().Unix()
	data, _ := proto.Marshal(&order)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundNewOrder, data)
	_, err := method.DoSend(db, senderAccBlock, 100100100)
	//fmt.Printf("err for send %s\n", err.Error())
	assert.True(t, err == nil)

	param := new(contracts.ParamDexSerializedData)
	contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundNewOrder, senderAccBlock.Data)

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now

	var appendedBlocks []*contracts.SendBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := contracts.GetFundFromStorage(db, userAddress)
	acc := dexFund.Accounts[0]

	assert.True(t, CheckBigEqualToInt(800, acc.Available))
	assert.True(t, CheckBigEqualToInt(2000, acc.Locked))
	assert.Equal(t, 1, len(appendedBlocks))

	param1 := new(contracts.ParamDexSerializedData)
	err = contracts.ABIDexTrade.UnpackMethod(param1, contracts.MethodNameDexTradeNewOrder, appendedBlocks[0].Data)
	order1 := &dexproto.Order{}
	proto.Unmarshal(param1.Data, order1)
	assert.True(t, CheckBigEqualToInt(60, order1.Amount))
	assert.Equal(t, order1.Status, uint32(dex.Pending))
}

func innerTestSettleOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	method := contracts.MethodDexFundSettleOrders{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = types.AddressDexTrade

	viteAction := dexproto.SettleAction{}
	viteAction.Address = userAddress.Bytes()
	viteAction.Token = VITE.Bytes()
	viteAction.DeduceLocked = big.NewInt(1000).Bytes()

	ethAction := dexproto.SettleAction{}
	ethAction.Address = userAddress.Bytes()
	ethAction.Token = ETH.Bytes()
	ethAction.IncAvailable = big.NewInt(30).Bytes()

	actions := dexproto.SettleActions{}
	actions.Actions = append(actions.Actions, &viteAction)
	actions.Actions = append(actions.Actions, &ethAction)
	data, _ := proto.Marshal(&actions)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundSettleOrders, data)
	_, err := method.DoSend(db, senderAccBlock, 100100100)
	//fmt.Printf("err %s\n", err.Error())
	assert.True(t, err == nil)

	receiveBlock := &ledger.AccountBlock{}
	_, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	//fmt.Printf("receive err %s\n", err.Error())
	dexFund, _ := contracts.GetFundFromStorage(db, userAddress)
	assert.Equal(t, 2, len(dexFund.Accounts))
	var ethAcc, viteAcc *dexproto.Account
	acc := dexFund.Accounts[0]
	if bytes.Equal(acc.Token, ETH.Bytes()) {
		ethAcc = dexFund.Accounts[0]
		viteAcc = dexFund.Accounts[1]
	} else {
		ethAcc = dexFund.Accounts[1]
		viteAcc = dexFund.Accounts[0]
	}
	assert.True(t, CheckBigEqualToInt(30, ethAcc.Available))
	assert.True(t, CheckBigEqualToInt(1000, viteAcc.Locked))
}

func initDexFundDatabase() *testDatabase {
	db := NewNoDatabase()
	db.addr = types.AddressDexFund
	return db
}

func depositCash(db *testDatabase, address types.Address, amount uint64, token types.TokenTypeId) {
	if _, ok := db.balanceMap[address]; !ok {
		db.balanceMap[address] = make(map[types.TokenTypeId]*big.Int)
	}
	db.balanceMap[address][token] = big.NewInt(0).SetUint64(amount)
}

func registerToken(db *testDatabase, token types.TokenTypeId) {
	tokenName := string(token.Bytes()[0:4])
	tokenSymbol := string(token.Bytes()[5:10])
	decimals := uint8(18)
	tokenData, _ := abi.ABIMintage.PackVariable(abi.VariableNameMintage, tokenName, tokenSymbol, big.NewInt(1e16), decimals, ledger.GenesisAccountAddress, big.NewInt(0), uint64(0))
	if _, ok := db.storageMap[types.AddressMintage]; !ok {
		db.storageMap[types.AddressMintage] = make(map[string][]byte)
	}
	mintageKey := string(cabi.GetMintageKey(token))
	db.storageMap[types.AddressMintage][mintageKey] = tokenData
}

func CheckBigEqualToInt(expected int, value []byte) bool {
	return new(big.Int).SetUint64(uint64(expected)).Cmp(new(big.Int).SetBytes(value)) == 0
}
