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
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"testing"
	"time"
)

func TestDexFund(t *testing.T) {
	ethTokenInfo := dexproto.TokenInfo{}
	ethTokenInfo.Decimals = ETH.decimals
	ethTokenInfo.Symbol = ETH.Symbol
	ethTokenInfo.Index = int32(ETH.Index)
	dex.QuoteTokenInfos[ETH.tokenId] = &dex.TokenInfo{ethTokenInfo}
	dex.QuoteTokenMinAmount[ETH.tokenId] = big.NewInt(6)
	db := initDexFundDatabase()
	userAddress, _ := types.BytesToAddress([]byte("123456789012345678901"))
	depositCash(db, userAddress, 3000, VITE.tokenId)
	registerToken(db, VITE)
	innerTestDepositAndWithdraw(t, db, userAddress)
	innerTestFundNewOrder(t, db, userAddress)
	innerTestVerifyBalance(t, db)
	//innerTestNewMarket(t, db)
	innerTestSettleOrder(t, db, userAddress)
}

func innerTestDepositAndWithdraw(t *testing.T, db *testDatabase, userAddress types.Address) {
	var err error
	//deposit
	depositMethod := contracts.MethodDexFundUserDeposit{}

	depositSendAccBlock := &ledger.AccountBlock{}
	depositSendAccBlock.AccountAddress = userAddress
	depositSendAccBlock.TokenId = VITE.tokenId
	depositSendAccBlock.Amount = big.NewInt(3000)

	depositSendAccBlock.Data, err = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserDeposit)
	err = depositMethod.DoSend(db, depositSendAccBlock)
	assert.True(t, err == nil)
	assert.True(t, bytes.Equal(depositSendAccBlock.TokenId.Bytes(), VITE.tokenId.Bytes()))
	assert.Equal(t, depositSendAccBlock.Amount.Uint64(), uint64(3000))

	depositReceiveBlock := &ledger.AccountBlock{}

	err = doDeposit(db, depositMethod, depositReceiveBlock, depositSendAccBlock)
	assert.True(t, err == nil)

	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress)
	assert.Equal(t, 1, len(dexFund.Accounts))
	acc := dexFund.Accounts[0]
	assert.True(t, bytes.Equal(acc.Token, VITE.tokenId.Bytes()))
	assert.True(t, checkBigEqualToInt(3000, acc.Available))
	assert.True(t, checkBigEqualToInt(0, acc.Locked))

	//withdraw
	withdrawMethod := contracts.MethodDexFundUserWithdraw{}

	withdrawSendAccBlock := &ledger.AccountBlock{}
	withdrawSendAccBlock.AccountAddress = userAddress
	withdrawSendAccBlock.Data, err = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserWithdraw, VITE.tokenId, big.NewInt(200))
	err = withdrawMethod.DoSend(db, withdrawSendAccBlock)
	assert.True(t, err == nil)

	withdrawReceiveBlock := &ledger.AccountBlock{}

	clearLogs(db)
	withdrawSendAccBlock.Data, err = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundUserWithdraw, VITE.tokenId, big.NewInt(200))
	appendedBlocks, _ := doWithdraw(db, withdrawMethod, withdrawReceiveBlock, withdrawSendAccBlock)
	dexFund, _ = dex.GetUserFundFromStorage(db, userAddress)
	acc = dexFund.Accounts[0]
	assert.True(t, checkBigEqualToInt(2800, acc.Available))
	assert.Equal(t, 1, len(appendedBlocks))

}

func innerTestNewMarket(t *testing.T, db *testDatabase) {
	userAddress1, _ := types.BytesToAddress([]byte("123456789012345678902"))
	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress1
	senderAccBlock.TokenId = ETH.tokenId
	senderAccBlock.Amount = new(big.Int).Div(dex.NewMarketFeeDividendAmount, big.NewInt(2))
	senderAccBlock.Data, _ = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewMarket, VITE.tokenId, ETH.tokenId)
	method := contracts.MethodDexFundNewMarket{}
	err := method.DoSend(db, senderAccBlock)
	assert.Equal(t, "token type of fee for create market not valid", err.Error())
	senderAccBlock.TokenId = VITE.tokenId
	err = method.DoSend(db, senderAccBlock)
	assert.Equal(t, "fee for create market not enough", err.Error())
	senderAccBlock.Amount = new(big.Int).Add(dex.NewMarketFeeAmount, dex.NewMarketFeeDividendAmount)

	receiveBlock := &ledger.AccountBlock{}
	err = doNewMarket(db, method, receiveBlock, senderAccBlock)
	assert.True(t, err == nil)
	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress1)
	acc := dexFund.Accounts[0]
	assert.True(t, dex.CmpForBigInt(dex.NewMarketFeeDividendAmount.Bytes(), acc.Available) == 0)

	marketInfo, ok := dex.GetMarketInfo(db, VITE.tokenId, ETH.tokenId)
	assert.True(t, ok)
	assert.True(t, bytes.Equal(marketInfo.Creator, userAddress1.Bytes()))
	assert.Equal(t, VITE.decimals, marketInfo.TradeTokenDecimals)
	assert.Equal(t, ETH.decimals, marketInfo.QuoteTokenDecimals)
	err = doNewMarket(db, method, receiveBlock, senderAccBlock)

	assert.Equal(t, dex.TradeMarketExistsError, err)
	userFees, _ := dex.GetUserFeesFromStorage(db, userAddress1.Bytes())
	assert.Equal(t, 1, len(userFees.Fees))
	assert.True(t, bytes.Equal(userFees.Fees[0].UserFees[0].Token, VITE.tokenId.Bytes()))
	assert.True(t, bytes.Equal(userFees.Fees[0].UserFees[0].Amount, dex.NewMarketFeeDividendAmount.Bytes()))
}

func innerTestVerifyBalance(t *testing.T, db *testDatabase) {
	resMap := dex.VerifyDexFundBalance(db)
	for _, v := range resMap.VerifyItems {
		if v.TokenId != vxTokenId {
			//fmt.Printf("token %s, balance %s, amount %s, ok %v\n", string(v.TokenId.Bytes()), v.Balance, v.Amount, v.Ok)
			assert.True(t, v.Ok)
		}
	}
}

func innerTestFundNewOrder(t *testing.T, db *testDatabase, userAddress types.Address) {
	registerToken(db, ETH)
	method := contracts.MethodDexFundNewOrder{}
	senderAccBlock := &ledger.AccountBlock{}
	senderAccBlock.AccountAddress = userAddress

	receiveBlock := &ledger.AccountBlock{}

	clearLogs(db)

	senderAccBlock.Data, _ = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewOrder, VITE.tokenId.Bytes(), ETH.tokenId.Bytes(), true, int8(dex.Limited), "0.3", big.NewInt(1000))
	//fmt.Printf("PackMethod err for send %s\n", err.Error())
	_, err := method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, err, dex.TradeMarketNotExistsError)

	innerTestNewMarket(t, db)
	clearLogs(db)

	_, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, err, dex.OrderAmountTooSmallErr)

	clearLogs(db)
	senderAccBlock.Data, _ = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundNewOrder, VITE.tokenId.Bytes(), ETH.tokenId.Bytes(), true, int8(dex.Limited), "0.3", big.NewInt(2000))
	var appendedBlocks []*ledger.AccountBlock
	appendedBlocks, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.True(t, err == nil)
	assert.Equal(t, 0, len(db.logList))

	dexFund, _ := dex.GetUserFundFromStorage(db, userAddress)
	acc := dexFund.Accounts[0]
	assert.Equal(t, 1, len(appendedBlocks))
	assert.True(t, checkBigEqualToInt(800, acc.Available))
	assert.True(t, checkBigEqualToInt(2000, acc.Locked))

	param1 := new(dex.ParamDexSerializedData)
	err = abi.ABIDexTrade.UnpackMethod(param1, abi.MethodNameDexTradeNewOrder, appendedBlocks[0].Data)
	order := &dex.Order{}
	err = order.DeSerialize(param1.Data)
	assert.Equal(t, err, nil)
	assert.True(t, checkBigEqualToInt(6, order.Amount))
	assert.Equal(t, order.Status, int32(dex.Pending))
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

	senderAccBlock.Data, _ = abi.ABIDexFund.PackMethod(abi.MethodNameDexFundSettleOrders, data)
	err := method.DoSend(db, senderAccBlock)
	assert.True(t, err == nil)

	receiveBlock := &ledger.AccountBlock{}

	clearLogs(db)

	_, err = method.DoReceive(db, receiveBlock, senderAccBlock, nil)
	assert.Equal(t, 0, len(db.logList))

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
	assert.True(t, checkBigEqualToInt(0, ethAcc.Locked))
	assert.True(t, checkBigEqualToInt(30, ethAcc.Available))
	assert.True(t, checkBigEqualToInt(900, viteAcc.Locked))
	assert.True(t, checkBigEqualToInt(900, viteAcc.Available))

	dexFee, _ := dex.GetCurrentFeeSumFromStorage(db, getConsensusReader()) // initDexFundDatabase snapshotBlock Height
	assert.Equal(t, 2, len(dexFee.Fees))
	if bytes.Equal(dexFee.Fees[0].Token, ETH.tokenId.Bytes()) {
		assert.True(t, checkBigEqualToInt(15, dexFee.Fees[0].Amount))
		assert.True(t, bytes.Equal(dex.NewMarketFeeDividendAmount.Bytes(), dexFee.Fees[1].Amount))
	} else {
		assert.True(t, checkBigEqualToInt(15, dexFee.Fees[1].Amount))
		assert.True(t, bytes.Equal(dex.NewMarketFeeDividendAmount.Bytes(), dexFee.Fees[0].Amount))
	}
}

func initDexFundDatabase() *testDatabase {
	db := newNoDatabase()
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
		uint8(25), //nodeCount
		int64(1),  //interval
		int64(3),  //perCount
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		helper.JoinBytes(helper.LeftPadBytes(new(big.Int).Mul(big.NewInt(1e6), util.AttovPerVite).Bytes(), helper.WordSize), helper.LeftPadBytes(ledger.ViteTokenId.Bytes(), helper.WordSize), helper.LeftPadBytes(big.NewInt(3600 * 24 * 90).Bytes(), helper.WordSize)),
		uint8(1),
		[]byte{},
		db.addr,
		big.NewInt(0),
		uint64(1))
	db.storageMap[types.AddressConsensusGroup][string(consensusGroupKey.Bytes())] = consensusGroupData
	dex.SetTimerTimestamp(db, time.Now().Unix())
	return db
}

func rollPeriod(db *testDatabase) {
	lastSnapshot := db.snapshotBlockList[len(db.snapshotBlockList)-1]
	t1 := time.Unix(int64(lastSnapshot.Timestamp.Unix()+75), 0)
	snapshot1 := &ledger.SnapshotBlock{Height: lastSnapshot.Height + 1, Timestamp: &t1, Hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)
}

func depositCash(db *testDatabase, address types.Address, amount uint64, token types.TokenTypeId) {
	if _, ok := db.balanceMap[address]; !ok {
		db.balanceMap[address] = make(map[types.TokenTypeId]*big.Int)
	}
	db.balanceMap[address][token] = big.NewInt(0).SetUint64(amount)
}

func registerToken(db *testDatabase, tokenInput tokenInfo) {
	token := dexproto.TokenInfo{}
	token.Symbol = string(tokenInput.tokenId[:4])
	token.Decimals = tokenInput.decimals
	tokenInfo := &dex.TokenInfo{token}
	dex.SaveTokenInfo(db, tokenInput.tokenId, tokenInfo)
}

func orderIdBytesFromInt(v int) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(v))
	return append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, bs...)
}

func doDeposit(db *testDatabase, depositMethod contracts.MethodDexFundUserDeposit, receiveBlock, sendBlock *ledger.AccountBlock) (err error) {
	if _, err = depositMethod.DoReceive(db, receiveBlock, sendBlock, nil); err == nil {
		addBalance(db, sendBlock.TokenId, sendBlock.Amount)
	}
	return err
}

func doWithdraw(db *testDatabase, withdrawMethod contracts.MethodDexFundUserWithdraw, receiveBlock, sendBlock *ledger.AccountBlock) (appendedBlocks []*ledger.AccountBlock, err error) {
	if appendedBlocks, err = withdrawMethod.DoReceive(db, receiveBlock, sendBlock, nil); err == nil {
		param := new(dex.ParamDexFundWithDraw)
		abi.ABIDexFund.UnpackMethod(param, abi.MethodNameDexFundUserWithdraw, sendBlock.Data)
		subBalance(db, param.Token, param.Amount)
	}
	return appendedBlocks, err
}

func doNewMarket(db *testDatabase, newMarketMethod contracts.MethodDexFundNewMarket, receiveBlock, sendBlock *ledger.AccountBlock) (err error) {
	if _, err = newMarketMethod.DoReceive(db, receiveBlock, sendBlock, nil); err == nil {
		addBalance(db, sendBlock.TokenId, sendBlock.Amount)
	}
	return err
}