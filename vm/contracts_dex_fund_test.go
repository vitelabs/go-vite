package vm

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"math/big"
	"testing"
	"time"
)


type tokenInfo struct {
	tokenId types.TokenTypeId
	decimals int32
}

var (
	ETH = tokenInfo{types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}, 12} //tradeToken
	VITE = tokenInfo{types.TokenTypeId{'V', 'I', 'T', 'E', ' ', 'T', 'O', 'K', 'E', 'N'}, 14} //quoteToken
)

func TestDexFund(t *testing.T) {
	db := initDexFundDatabase()
	userAddress, _ := types.BytesToAddress([]byte("12345678901234567890"))
	depositCash(db, userAddress, 3000, VITE.tokenId)
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

	depositSendAccBlock.TokenId = ETH.tokenId
	depositSendAccBlock.Amount = big.NewInt(100)

	depositSendAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	_, err = depositMethod.DoSend(db, depositSendAccBlock, 100010001000)
	assert.True(t, err != nil)
	assert.True(t, bytes.Equal([]byte(err.Error()), []byte("token is invalid")))

	depositSendAccBlock.TokenId = VITE.tokenId
	depositSendAccBlock.Amount = big.NewInt(3000)

	depositSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	_, err = depositMethod.DoSend(db, depositSendAccBlock, 100010001000)
	assert.True(t, err == nil)
	assert.True(t, bytes.Equal(depositSendAccBlock.TokenId.Bytes(), VITE.tokenId.Bytes()))
	assert.Equal(t, depositSendAccBlock.Amount.Uint64(), uint64(3000))

	depositReceiveBlock := &ledger.AccountBlock{}

	_, err = depositMethod.DoReceive(db, depositReceiveBlock, depositSendAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress)
	assert.Equal(t, 1, len(dexFund.Accounts))
	acc := dexFund.Accounts[0]
	assert.True(t, bytes.Equal(acc.Token, VITE.tokenId.Bytes()))
	assert.True(t, CheckBigEqualToInt(3000, acc.Available))
	assert.True(t, CheckBigEqualToInt(0, acc.Locked))

	//withdraw
	withdrawMethod := contracts.MethodDexFundUserWithdraw{}

	withdrawSendAccBlock := &ledger.AccountBlock{}
	withdrawSendAccBlock.AccountAddress = userAddress
	withdrawSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, VITE.tokenId, big.NewInt(200))
	_, err = withdrawMethod.DoSend(db, withdrawSendAccBlock, 100010001000)
	assert.True(t, err == nil)

	withdrawReceiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	withdrawReceiveBlock.Timestamp = &now

	withdrawSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, VITE.tokenId, big.NewInt(200))
	appendedBlocks, _ := withdrawMethod.DoReceive(db, withdrawReceiveBlock, withdrawSendAccBlock)

	dexFund, _ = dex.GetUserFundFromStorage(db, userAddress)
	acc = dexFund.Accounts[0]
	assert.True(t, CheckBigEqualToInt(2800, acc.Available))
	assert.Equal(t, 1, len(appendedBlocks))

}

func innerTestFundNewOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	registerToken(db, ETH)

	method := contracts.MethodDexFundNewOrder{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress
	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundNewOrder, orderIdBytesFromInt(1), VITE.tokenId.Bytes(), ETH.tokenId.Bytes(), true, uint32(dex.Limited), "0.3", big.NewInt(2000))
	//fmt.Printf("PackMethod err for send %s\n", err.Error())
	_, err := method.DoSend(db, senderAccBlock, 100100100)
	assert.True(t, err == nil)

	param := new(contracts.ParamDexFundNewOrder)
	err = contracts.ABIDexFund.UnpackMethod(param, contracts.MethodNameDexFundNewOrder, senderAccBlock.Data)
	assert.True(t, err == nil)
	//fmt.Printf("UnpackMethod err for send %s\n", err.Error())

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now
	receiveBlock.SnapshotHash = types.DataHash([]byte{10, 1})

	var appendedBlocks []*contracts.SendBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress)
	acc := dexFund.Accounts[0]

	assert.True(t, CheckBigEqualToInt(800, acc.Available))
	assert.True(t, CheckBigEqualToInt(2000, acc.Locked))
	assert.Equal(t, 1, len(appendedBlocks))

	param1 := new(contracts.ParamDexSerializedData)
	err = contracts.ABIDexTrade.UnpackMethod(param1, contracts.MethodNameDexTradeNewOrder, appendedBlocks[0].Data)
	order1 := &dexproto.Order{}
	proto.Unmarshal(param1.Data, order1)
	assert.True(t, CheckBigEqualToInt(6, order1.Amount))
	assert.Equal(t, order1.Status, int32(dex.Pending))
}

func innerTestSettleOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	method := contracts.MethodDexFundSettleOrders{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = types.AddressDexTrade

	viteAction := dexproto.FundSettle{}
	viteAction.Address = userAddress.Bytes()
	viteAction.Token = VITE.tokenId.Bytes()
	viteAction.ReduceLocked = big.NewInt(1000).Bytes()
	viteAction.ReleaseLocked = big.NewInt(100).Bytes()

	ethAction := dexproto.FundSettle{}
	ethAction.Address = userAddress.Bytes()
	ethAction.Token = ETH.tokenId.Bytes()
	ethAction.IncAvailable = big.NewInt(30).Bytes()

	feeAction := dexproto.FeeSettle{}
	feeAction.Token = ETH.tokenId.Bytes()
	feeAction.Amount = big.NewInt(15).Bytes()

	actions := dexproto.SettleActions{}
	actions.FundActions = append(actions.FundActions, &viteAction)
	actions.FundActions = append(actions.FundActions, &ethAction)
	actions.FeeActions = append(actions.FeeActions, &feeAction)
	data, _ := proto.Marshal(&actions)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundSettleOrders, data)
	_, err := method.DoSend(db, senderAccBlock, 100100100)
	//fmt.Printf("err %s\n", err.Error())
	assert.True(t, err == nil)

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now
	receiveBlock.SnapshotHash = types.DataHash([]byte{10, 1})
	_, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	//fmt.Printf("receive err %s\n", err.Error())
	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress)
	assert.Equal(t, 2, len(dexFund.Accounts))
	var ethAcc, viteAcc *dexproto.Account
	acc := dexFund.Accounts[0]
	if bytes.Equal(acc.Token, ETH.tokenId.Bytes()) {
		ethAcc = dexFund.Accounts[0]
		viteAcc = dexFund.Accounts[1]
	} else {
		ethAcc = dexFund.Accounts[1]
		viteAcc = dexFund.Accounts[0]
	}
	assert.True(t, CheckBigEqualToInt(0, ethAcc.Locked))
	assert.True(t, CheckBigEqualToInt(30, ethAcc.Available))
	assert.True(t, CheckBigEqualToInt(900, viteAcc.Locked))
	assert.True(t, CheckBigEqualToInt(900, viteAcc.Available))

	dexFee, err := dex.GetFeeFromStorage(db, 123) // initDexFundDatabase snapshotBlock Height
	assert.Equal(t, 1, len(dexFee.Fees))
	assert.False(t, dexFee.Divided)
	feeAcc := dexFee.Fees[0]
	assert.True(t, CheckBigEqualToInt(15, feeAcc.Amount))
}

func initDexFundDatabase() *testDatabase {
	db := NewNoDatabase()
	db.addr = types.AddressDexFund
	t1 := time.Unix(1536214502, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 123, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)
	return db
}

func depositCash(db *testDatabase, address types.Address, amount uint64, token types.TokenTypeId) {
	if _, ok := db.balanceMap[address]; !ok {
		db.balanceMap[address] = make(map[types.TokenTypeId]*big.Int)
	}
	db.balanceMap[address][token] = big.NewInt(0).SetUint64(amount)
}

func registerToken(db *testDatabase, token tokenInfo) {
	tokenName := string(token.tokenId.Bytes()[0:4])
	tokenSymbol := string(token.tokenId.Bytes()[5:10])
	decimals := uint8(token.decimals)
	tokenData, _ := abi.ABIMintage.PackVariable(abi.VariableNameMintage, tokenName, tokenSymbol, big.NewInt(1e16), decimals, ledger.GenesisAccountAddress, big.NewInt(0), uint64(0))
	if _, ok := db.storageMap[types.AddressMintage]; !ok {
		db.storageMap[types.AddressMintage] = make(map[string][]byte)
	}
	mintageKey := string(cabi.GetMintageKey(token.tokenId))
	db.storageMap[types.AddressMintage][mintageKey] = tokenData
}

func CheckBigEqualToInt(expected int, value []byte) bool {
	return new(big.Int).SetUint64(uint64(expected)).Cmp(new(big.Int).SetBytes(value)) == 0
}

func orderIdBytesFromInt(v int) []byte {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(v))
	return append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, bs...)
}