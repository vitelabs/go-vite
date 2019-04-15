package vm

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strconv"
	"testing"
	"time"
)


type tokenInfo struct {
	tokenId types.TokenTypeId
	decimals int32
}

var (
	ETH = tokenInfo{types.TokenTypeId{'E', 'T', 'H', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}, 12} //tradeToken
	VITE = tokenInfo{ledger.ViteTokenId, 14} //quoteToken
)

func TestDexFund(t *testing.T) {
	dex.QuoteTokenMinAmount[ETH.tokenId] = big.NewInt(6)
	db := initDexFundDatabase()
	userAddress, _ := types.BytesToAddress([]byte("12345678901234567890"))
	depositCash(db, userAddress, 3000, VITE.tokenId)
	registerToken(db, VITE)
	innerTestDepositAndWithdraw(t, db, userAddress)
	innerTestFundNewOrder(t, db, userAddress)
	//innerTestNewMarket(t, db)
	innerTestSettleOrder(t, db, userAddress)
}

func innerTestDepositAndWithdraw(t *testing.T, db *testDatabase, userAddress types.Address) {
	var err error
	//deposit
	depositMethod := contracts.MethodDexFundUserDeposit{}

	depositSendAccBlock := &ledger.AccountBlock{}
	depositSendAccBlock.AccountAddress = userAddress

	depositSendAccBlock.TokenId = ETH.tokenId
	depositSendAccBlock.Amount = big.NewInt(100)

	depositSendAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	err = depositMethod.DoSend(db, depositSendAccBlock)
	assert.True(t, err != nil)
	assert.True(t, bytes.Equal([]byte(err.Error()), []byte("token is invalid")))

	depositSendAccBlock.TokenId = VITE.tokenId
	depositSendAccBlock.Amount = big.NewInt(3000)

	depositSendAccBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	err = depositMethod.DoSend(db, depositSendAccBlock)
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
	err = withdrawMethod.DoSend(db, withdrawSendAccBlock)
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

func innerTestNewMarket(t *testing.T, db *testDatabase) {
	userAddress1, _ := types.BytesToAddress([]byte("12345678901234567891"))
	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress1
	senderAccBlock.TokenId = ETH.tokenId
	senderAccBlock.Amount = new(big.Int).Div(dex.NewMarketFeeDividendAmount, big.NewInt(2))
	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundNewMarket, VITE.tokenId, ETH.tokenId)
	method := contracts.MethodDexFundNewMarket{}
	err := method.DoSend(db, senderAccBlock)
	assert.Equal(t, "token type of fee for create market not valid", err.Error())
	senderAccBlock.TokenId = VITE.tokenId
	err = method.DoSend(db, senderAccBlock)
	assert.Equal(t, "fee for create market not enough", err.Error())
	senderAccBlock.Amount = dex.NewMarketFeeAmount
	senderAccBlock.Amount = new(big.Int).Add(dex.NewMarketFeeAmount, dex.NewMarketFeeDividendAmount)

	receiveBlock := &ledger.AccountBlock{}
	_, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress1)
	acc := dexFund.Accounts[0]
	assert.True(t, dex.CmpForBigInt(dex.NewMarketFeeDividendAmount.Bytes(), acc.Available) == 0)

	marketInfo, err := dex.GetMarketInfo(db, VITE.tokenId, ETH.tokenId)
	assert.True(t, marketInfo != nil)
	assert.True(t, bytes.Equal(marketInfo.Creator, userAddress1.Bytes()))
	assert.Equal(t, VITE.decimals, marketInfo.TradeTokenDecimals)
	assert.Equal(t, ETH.decimals, marketInfo.QuoteTokenDecimals)
	_, err = method.DoReceive(db, receiveBlock, senderAccBlock)

	assert.Equal(t, dex.TradeMarketExistsError, err)
	userFees, _ := dex.GetUserFeesFromStorage(db, userAddress1.Bytes())
	assert.Equal(t, 1, len(userFees.Fees))
	assert.True(t, bytes.Equal(userFees.Fees[0].UserFees[0].Token, VITE.tokenId.Bytes()))
	assert.True(t, bytes.Equal(userFees.Fees[0].UserFees[0].Amount, dex.NewMarketFeeDividendAmount.Bytes()))
}

func innerTestFundNewOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	registerToken(db, ETH)
	method := contracts.MethodDexFundNewOrder{}
	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress

	receiveBlock := &ledger.AccountBlock{}
	now := time.Now()
	receiveBlock.Timestamp = &now
	receiveBlock.SnapshotHash = types.DataHash([]byte{10, 1})

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundNewOrder, orderIdBytesFromInt(1), VITE.tokenId.Bytes(), ETH.tokenId.Bytes(), true, uint32(dex.Limited), "0.3", big.NewInt(1000))
	//fmt.Printf("PackMethod err for send %s\n", err.Error())
	_, err := method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.Equal(t, err, nil)
	eventLen := len(db.logList)
	invalidMarketEvent := dex.NewOrderFailEvent{}
	assert.Equal(t, invalidMarketEvent.GetTopicId().Bytes(), db.logList[eventLen - 1].Topics[0].Bytes())
	invalidMarketEvent, _ = invalidMarketEvent.FromBytes(db.logList[eventLen - 1].Data).(dex.NewOrderFailEvent)
	assert.Equal(t, strconv.Itoa(dex.TradeMarketNotExistsFail), invalidMarketEvent.ErrCode)

	innerTestNewMarket(t, db)

	_, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	eventLen = len(db.logList)
	tooSmallAmountEvent := dex.NewOrderFailEvent{}
	assert.Equal(t, tooSmallAmountEvent.GetTopicId().Bytes(), db.logList[eventLen - 1].Topics[0].Bytes())
	tooSmallAmountEvent, _ = tooSmallAmountEvent.FromBytes(db.logList[eventLen - 1].Data).(dex.NewOrderFailEvent)
	assert.Equal(t, strconv.Itoa(dex.OrderAmountTooSmallFail), tooSmallAmountEvent.ErrCode)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundNewOrder, orderIdBytesFromInt(1), VITE.tokenId.Bytes(), ETH.tokenId.Bytes(), true, uint32(dex.Limited), "0.3", big.NewInt(2000))
	var appendedBlocks []*contracts.SendBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	assert.Equal(t, eventLen, len(db.logList))
	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress)
	acc := dexFund.Accounts[0]

	assert.True(t, CheckBigEqualToInt(800, acc.Available))
	assert.True(t, CheckBigEqualToInt(2000, acc.Locked))
	assert.Equal(t, 1, len(appendedBlocks))

	param1 := new(dex.ParamDexSerializedData)
	err = contracts.ABIDexTrade.UnpackMethod(param1, contracts.MethodNameDexTradeNewOrder, appendedBlocks[0].Data)
	order1 := &dexproto.OrderInfo{}
	proto.Unmarshal(param1.Data, order1)
	assert.True(t, CheckBigEqualToInt(6, order1.Order.Amount))
	assert.Equal(t, order1.Order.Status, int32(dex.Pending))
}

func innerTestSettleOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	method := contracts.MethodDexFundSettleOrders{}

	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = types.AddressDexTrade

	viteFundSettle := &dexproto.FundSettle{}
	viteFundSettle.Token = VITE.tokenId.Bytes()
	viteFundSettle.ReduceLocked = big.NewInt(1000).Bytes()
	viteFundSettle.ReleaseLocked = big.NewInt(100).Bytes()

	ethFundSettle := &dexproto.FundSettle{}
	ethFundSettle.Token = ETH.tokenId.Bytes()
	ethFundSettle.IncAvailable = big.NewInt(30).Bytes()

	fundAction := &dexproto.UserFundSettle{}
	fundAction.Address = userAddress.Bytes()
	fundAction.FundSettles = append(fundAction.FundSettles, viteFundSettle, ethFundSettle)

	feeAction := dexproto.FeeSettle{}
	feeAction.Token = ETH.tokenId.Bytes()
	userFeeSettle := &dexproto.UserFeeSettle{}
	userFeeSettle.Address = userAddress.Bytes()
	userFeeSettle.Amount = big.NewInt(15).Bytes()
	feeAction.UserFeeSettles = append(feeAction.UserFeeSettles, userFeeSettle)

	actions := dexproto.SettleActions{}
	actions.FundActions = append(actions.FundActions, fundAction)
	actions.FeeActions = append(actions.FeeActions, &feeAction)
	data, _ := proto.Marshal(&actions)

	senderAccBlock.Data, _ = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundSettleOrders, data)
	err := method.DoSend(db, senderAccBlock)
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

	dexFee, err := dex.GetCurrentFeeSumFromStorage(db) // initDexFundDatabase snapshotBlock Height
	assert.Equal(t, 2, len(dexFee.Fees))
	if bytes.Equal(dexFee.Fees[0].Token, ETH.tokenId.Bytes()) {
		assert.True(t, CheckBigEqualToInt(15, dexFee.Fees[0].Amount))
		assert.True(t, bytes.Equal(dex.NewMarketFeeDividendAmount.Bytes(), dexFee.Fees[1].Amount))
	} else {
		assert.True(t, CheckBigEqualToInt(15, dexFee.Fees[1].Amount))
		assert.True(t, bytes.Equal(dex.NewMarketFeeDividendAmount.Bytes(), dexFee.Fees[0].Amount))
	}

}

func initDexFundDatabase() *testDatabase {
	db := NewNoDatabase()
	db.addr = types.AddressDexFund
	t1 := time.Unix(1536214502, 0)
	snapshot1 := &ledger.SnapshotBlock{Height: 123, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)

	db.storageMap[types.AddressConsensusGroup] = make(map[string][]byte)
	consensusGroupKey, _ := types.BytesToHash(abi.GetConsensusGroupKey(types.SNAPSHOT_GID))

	//	planInterval = uint64(info.Interval) * uint64(info.NodeCount) * uint64(info.PerCount)
	// 	75 = 1 * 25 * 3
	//	subSec := int64(t.Sub(self.genesisTime).Seconds())
	//	i := uint64(subSec) / self.PlanInterval
	consensusGroupData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),//nodeCount
		int64(1),//interval
		int64(3),//perCount
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(3600*24*90).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		db.addr,
		big.NewInt(0),
		uint64(1))
	db.storageMap[types.AddressConsensusGroup][string(consensusGroupKey.Bytes())] = consensusGroupData

	return db
}

func rollPeriod(db *testDatabase) {
	lastSnapshot := db.snapshotBlockList[len(db.snapshotBlockList) - 1]
	t1 := time.Unix(int64(lastSnapshot.Timestamp.Unix() + 75), 0)
	snapshot1 := &ledger.SnapshotBlock{Height: lastSnapshot.Height + 1, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)
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
	binary.BigEndian.PutUint32(bs, uint32(v))
	return append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, bs...)
}