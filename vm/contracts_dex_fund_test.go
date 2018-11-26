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

var (
	VITE = types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}
	ETH  = types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}
	NANO = types.TokenTypeId{'N', 'A', 'N', 'O', ' ', 'T', 'O', 'K', 'E', 'N'}
)

type vmMockVmCtx struct {
	appendedBlocks []*vm_context.VmAccountBlock
}

func(ctx *vmMockVmCtx) AppendBlock(block *vm_context.VmAccountBlock) {
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
	depositSenderVmBlock := &vm_context.VmAccountBlock{}
	depositSenderVmBlock.VmContext = db

	depositSendAccBlock := &ledger.AccountBlock{}
	depositSenderVmBlock.AccountBlock = depositSendAccBlock

	depositSendAccBlock.AccountAddress = userAddress

	depositSendAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit, userAddress, ETH, big.NewInt(100))
	_, err = depositMethod.DoSend(&VmContext{}, depositSenderVmBlock, 100010001000)
	assert.True(t, err != nil)
	assert.True(t, bytes.Equal([]byte(err.Error()), []byte("token is invalid")))

	depositSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit, userAddress, VITE, big.NewInt(3000))
	_, err = depositMethod.DoSend(&VmContext{}, depositSenderVmBlock, 100010001000)
	assert.True(t, err == nil)
	assert.True(t, bytes.Equal(depositSendAccBlock.TokenId.Bytes(), VITE.Bytes()))
	assert.Equal(t, depositSendAccBlock.Amount.Uint64(), uint64(3000))
	assert.True(t, bytes.Equal(depositSendAccBlock.ToAddress.Bytes(), contracts.AddressDexFund.Bytes()))

	depositReceiveVmBlock := &vm_context.VmAccountBlock{}
	depositReceiveVmBlock.VmContext = db
	err = depositMethod.DoReceive(&VmContext{}, depositReceiveVmBlock, depositSendAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := contracts.GetFundFromStorage(db, userAddress)
	assert.Equal(t, 1, len(dexFund.Accounts))
	acc := dexFund.Accounts[0]
	assert.True(t, bytes.Equal(acc.Token, VITE.Bytes()))
	assert.True(t, CheckBigEqualToInt(3000, acc.Available))
	assert.True(t, CheckBigEqualToInt(0, acc.Locked))


	//withdraw
	withdrawMethod := contracts.MethodDexFundUserWithdraw{}
	withdrawSenderVmBlock := &vm_context.VmAccountBlock{}
	withdrawSenderVmBlock.VmContext = db

	withdrawSendAccBlock := &ledger.AccountBlock{}
	withdrawSenderVmBlock.AccountBlock = withdrawSendAccBlock

	withdrawSendAccBlock.AccountAddress = userAddress

	withdrawSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, userAddress, VITE, big.NewInt(200))
	_, err = withdrawMethod.DoSend(&VmContext{}, withdrawSenderVmBlock, 100010001000)
	assert.True(t, err == nil)

	withdrawReceiveVmBlock := &vm_context.VmAccountBlock{}
	withdrawReceiveVmBlock.VmContext = db
	withdrawReceiveVmBlock.AccountBlock = &ledger.AccountBlock{}
	now := time.Now()
	withdrawReceiveVmBlock.AccountBlock.Timestamp = &now

	withdrawSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, userAddress, VITE, big.NewInt(200))
	context := &vmMockVmCtx{}
	withdrawMethod.DoReceive(context, withdrawReceiveVmBlock, withdrawSendAccBlock)

	dexFund, _ = contracts.GetFundFromStorage(db, userAddress)
	acc = dexFund.Accounts[0]
	assert.True(t, CheckBigEqualToInt(2800, acc.Available))
	assert.Equal(t, 1, len(context.appendedBlocks))

}

func innerTestFundNewOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	registerToken(db, ETH)

	method := contracts.MethodDexFundNewOrder{}
	senderVmBlock := &vm_context.VmAccountBlock{}
	senderVmBlock.VmContext = db

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress
	senderVmBlock.AccountBlock = senderAccBlock
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
	order.Status =  dex.FullyExecuted
	order.Timestamp = time.Now().UnixNano()/1000
	data, _ := proto.Marshal(&order)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundNewOrder, data)
	_, err := method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	//fmt.Printf("err for send %s\n", err.Error())
	assert.True(t, err == nil)

	param := new(contracts.ParamDexSerializedData)
	contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundNewOrder, senderVmBlock.AccountBlock.Data)
	order1 := &dexproto.Order{}
	proto.Unmarshal(param.Data, order1)
	assert.True(t, CheckBigEqualToInt(60, order1.Amount))
	assert.Equal(t, order1.Status, uint32(dex.Pending))

	receiveVmBlock := &vm_context.VmAccountBlock{}
	receiveVmBlock.VmContext = db
	receiveVmBlock.AccountBlock = &ledger.AccountBlock{}
	now := time.Now()
	receiveVmBlock.AccountBlock.Timestamp = &now

	context := &vmMockVmCtx{}

	err = method.DoReceive(context, receiveVmBlock, senderAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := contracts.GetFundFromStorage(db, userAddress)
	acc := dexFund.Accounts[0]
	assert.True(t, CheckBigEqualToInt(800, acc.Available))
	assert.True(t, CheckBigEqualToInt(2000, acc.Locked))
	assert.Equal(t, 1, len(context.appendedBlocks))
}


func innerTestSettleOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	method := contracts.MethodDexFundSettleOrders{}
	senderVmBlock := &vm_context.VmAccountBlock{}
	senderVmBlock.VmContext = db

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = contracts.AddressDexTrade
	senderVmBlock.AccountBlock = senderAccBlock

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
	data,_ := proto.Marshal(&actions)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundSettleOrders, data)
	_, err := method.DoSend(&VmContext{}, senderVmBlock, 100100100)
	//fmt.Printf("err %s\n", err.Error())
	assert.True(t, err == nil)

	receiveVmBlock := &vm_context.VmAccountBlock{}
	receiveVmBlock.VmContext = db

	err = method.DoReceive(&vmMockVmCtx{}, receiveVmBlock, senderAccBlock)
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

func initDexFundDatabase()  *testDatabase {
	db := NewNoDatabase()
	db.addr = contracts.AddressDexFund
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
	tokenData, _ := contracts.ABIMintage.PackVariable(contracts.VariableNameMintage, tokenName, tokenSymbol, big.NewInt(1e16), decimals, ledger.GenesisAccountAddress, big.NewInt(0), uint64(0))
	if _, ok := db.storageMap[contracts.AddressMintage]; !ok {
		db.storageMap[contracts.AddressMintage] = make(map[string][]byte)
	}
	mintageKey := string(contracts.GetMintageKey(token))
	db.storageMap[contracts.AddressMintage][mintageKey] = tokenData
}


func CheckBigEqualToInt(expected int, value []byte) bool {
	return new(big.Int).SetUint64(uint64(expected)).Cmp(new(big.Int).SetBytes(value)) == 0
}