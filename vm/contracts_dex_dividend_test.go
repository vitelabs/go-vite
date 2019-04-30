package vm

import (
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"testing"
)

var (
 	vxTokenId = types.TokenTypeId{'V', 'X', ' ', ' ', ' ', 'T', 'O', 'K', 'E', 'N'}
	vxTokenInfo = tokenInfo{vxTokenId, 5}

	userAddress0, _ = types.BytesToAddress([]byte("123456789012345678901"))
	userAddress1, _ = types.BytesToAddress([]byte("123456789012345678902"))
	userAddress2, _ = types.BytesToAddress([]byte("123456789012345678903"))
)

func TestDexDividend(t *testing.T) {
	db := initDexFundDatabase()
	registerToken(db, vxTokenInfo)
	rollPeriod(db)
	dex.VxTokenBytes = vxTokenId.Bytes()
	dex.VxDividendThreshold = big.NewInt(20)
	innerTestVxAccUpdate(t, db)
	innerTestFeeDividend(t, db)
	innerTestMinedVxForFee(t, db)
	innerTestVerifyBalance(t, db)
}

func innerTestVxAccUpdate(t *testing.T, db *testDatabase) {
	err := deposit(db, userAddress0, vxTokenId, 10)
	assert.True(t, err == nil)
	vxSumFunds, err := dex.GetVxSumFundsFromDb(db)
	assert.True(t, err == nil)
	assert.True(t, vxSumFunds == nil)

	user0VxFunds, err := dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.True(t, err == nil)
	assert.True(t, user0VxFunds == nil)

	deposit(db, userAddress0, vxTokenId, 20)
	vxSumFunds = checkVxSumLen(t, db, 1)
	assert.Equal(t, uint64(1), vxSumFunds.Funds[0].Period)
	assert.True(t, CheckBigEqualToInt(30, vxSumFunds.Funds[0].Amount))

	user0VxFunds, err = dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.Equal(t, 1, len(user0VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(30, user0VxFunds.Funds[0].Amount))

	periodId := dex.GetCurrentPeriodIdFromStorage(db, getConsensusReader())
	assert.Equal(t, uint64(1), periodId)
	err = settleFee(db, userAddress0, ETH.tokenId, 60)
	settleFee(db, userAddress0, VITE.tokenId, 30)
	settleFee(db, userAddress1, ETH.tokenId, 30)
	//periodId = 1
	// vxSumFunds 1 -> 30
	// userAddress0 vxFund 1 -> 30
	// feeSum 1 -> [ETH : 90, VITE 30]
	// userAddress0 userFees 1 -> [ETH : 60, VITE : 30]
	// userAddress1 userFees 1 -> [ETH : 30]

	rollPeriod(db)
	periodId = dex.GetCurrentPeriodIdFromStorage(db, getConsensusReader())
	assert.True(t, err == nil)
	assert.Equal(t, uint64(2), periodId)

	deposit(db, userAddress0, vxTokenId, 2)
	deposit(db, userAddress1, vxTokenId, 23)
	vxSumFunds, err = dex.GetVxSumFundsFromDb(db)
	assert.Equal(t, uint64(1), vxSumFunds.Funds[0].Period)
	assert.Equal(t, uint64(2), vxSumFunds.Funds[1].Period)

	assert.Equal(t, 2, len(vxSumFunds.Funds))
	assert.True(t, CheckBigEqualToInt(30, vxSumFunds.Funds[0].Amount))
	assert.True(t, CheckBigEqualToInt(55, vxSumFunds.Funds[1].Amount))

	user0VxFunds, err = dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.Equal(t, 2, len(user0VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(30, user0VxFunds.Funds[0].Amount))
	assert.True(t, CheckBigEqualToInt(32, user0VxFunds.Funds[1].Amount))

	user1VxFunds, err := dex.GetVxFundsFromStorage(db, userAddress1.Bytes())
	assert.Equal(t, 1, len(user1VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(23, user1VxFunds.Funds[0].Amount))

	settleFee(db, userAddress0, VITE.tokenId, 30)
	settleFee(db, userAddress1, VITE.tokenId, 30)
	settleFee(db, userAddress2, ETH.tokenId, 30)
	// periodId = 2
	// vxSumFunds 1 -> 30, 2 -> 55
	// userAddress0 vxFund 1 -> 30, 2 -> 32
	// userAddress1 vxFund 2 -> 23
	// feeSum 1 -> [ETH : 90, VITE 30], 2 -> [ETH : 30, VITE : 60]
	// userAddress0 userFees 1 -> [ETH : 60, VITE 30], 2 -> [VITE : 30]
	// userAddress1 userFees 1 -> [ETH : 30], 2 -> [VITE : 30]
	// userAddress2 userFees 2 -> [ETH : 30]

	rollPeriod(db)
	withdraw(db, userAddress0, vxTokenId, 12)	//20

	vxSumFunds = checkVxSumLen(t, db, 3)
	assert.True(t, CheckBigEqualToInt(43, vxSumFunds.Funds[2].Amount))

	user0VxFunds, err = dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.Equal(t, 3, len(user0VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(20, user0VxFunds.Funds[2].Amount))
	// periodId = 3
	// vxSumFunds 1 -> 30, 2 -> 55, 3 -> 43
	// userAddress0 vxFund 1 -> 30, 2 -> 32, 3 -> 20
	// userAddress1 vxFund 2 -> 23

	rollPeriod(db)
	withdraw(db, userAddress0, vxTokenId, 2) // 18
	withdraw(db, userAddress1, vxTokenId, 6) // 17

	vxSumFunds = checkVxSumLen(t, db, 4)
	assert.True(t, CheckBigEqualToInt(0, vxSumFunds.Funds[3].Amount))

	user0VxFunds, err = dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.Equal(t, 4, len(user0VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(18, user0VxFunds.Funds[3].Amount))

	user1VxFunds, _ = dex.GetVxFundsFromStorage(db, userAddress1.Bytes())
	assert.Equal(t, 2, len(user1VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(23, user1VxFunds.Funds[0].Amount))
	assert.True(t, CheckBigEqualToInt(17, user1VxFunds.Funds[1].Amount))

	settleFee(db, userAddress0, VITE.tokenId, 13)
	settleFee(db, userAddress1, VITE.tokenId, 17)
	settleFee(db, userAddress2, ETH.tokenId, 19)
	// periodId = 4
	// vxSumFunds 1 -> 30, 2 -> 55, 3 -> 43, 4 -> 0
	// userAddress0 vxFund 1 -> 30, 2 -> 32, 3 -> 20, 4 -> 18
	// userAddress1 vxFund 2 -> 23, 4 -> 17
	// feeSum 1 -> [ETH : 90, VITE 30], 2 -> [ETH : 30, VITE 60], 4 -> [ETH : 19, VITE : 30]
	// userAddress0 userFees 1 -> [ETH : 60, VITE 30], 2 -> [VITE : 30], 4 -> [VITE : 13]
	// userAddress1 userFees 1 -> [ETH : 30], 2 -> [VITE 30], 4 -> [VITE : 17]
	// userAddress2 userFees 2 -> [ETH : 30], 4 -> [ETH : 19]

	rollPeriod(db)
	settleFund(db, userAddress0, vxTokenId, 1) //19
	settleFund(db, userAddress1, vxTokenId, 2)       //19
	checkVxSumLen(t, db, 4)
	checkAccount(t, db, userAddress0, vxTokenId,19)
	checkAccount(t, db, userAddress1, vxTokenId,19)

	user0VxFunds, err = dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.Equal(t, 4, len(user0VxFunds.Funds))
	// periodId = 5
	// vxSumFunds 1 -> 30, 2 -> 32, 3 -> 20, 4 -> 0
	// userAddress0 vxFund 1 -> 30, 2 -> 32, 3 -> 20, 4 -> 18, 5 -> 19
	// userAddress1 vxFund 2 -> 23, 4 -> 17

	rollPeriod(db)
	settleFund(db, userAddress1, vxTokenId, 2) //21
	vxSumFunds = checkVxSumLen(t, db, 5)
	checkAccount(t, db, userAddress1, vxTokenId,21)
	assert.True(t, CheckBigEqualToInt(21, vxSumFunds.Funds[4].Amount))

	user1VxFunds, _ = dex.GetVxFundsFromStorage(db, userAddress1.Bytes())
	assert.Equal(t, 3, len(user1VxFunds.Funds))
	assert.True(t, CheckBigEqualToInt(21, user1VxFunds.Funds[2].Amount))
	checkVxSumLen(t, db, 5)
	// periodId = 6
	// vxSumFunds 1 -> 30, 2 -> 32, 3 -> 20, 4 -> 0, 6 -> 21
	// userAddress0 vxFund 1 -> 30, 2 -> 32, 3 -> 20, 4 -> 18
	// userAddress1 vxFund 2 -> 23, 4 -> 17, 6 -> 21
	// feeSum 1 -> [ETH : 90, VITE 30], 2 -> [ETH : 30, VITE 60], 4 -> [ETH : 19, VITE : 30]
	// userAddress0 userFees 1 -> [ETH : 60, VITE 30], 2 -> [VITE : 30], 4 -> [VITE : 13]
	// userAddress1 userFees 1 -> [ETH : 30], 2 -> [VITE 30], 4 -> [VITE : 17]
	// userAddress2 userFees 2 -> [ETH : 30], 4 -> [ETH : 19]
}

func innerTestFeeDividend(t *testing.T, db *testDatabase) {
	err := feeDividend(db, 2)
	// vxSumFunds 3 -> 20, 4 -> 0, 6 -> 21
	// userAddress0 vxFund 3 -> 20, 4 -> 18
	// userAddress1 vxFund 2 -> 23, 4 -> 17, 6 -> 21
	// feeSum 4 -> [ETH : 19, VITE : 30]
	// userAddress0 userFees 4 -> [VITE : 13]
	// userAddress1 userFees 4 -> [VITE : 17]
	// userAddress2 userFees 4 -> [ETH : 19]
	assert.True(t, err == nil)
	assert.Equal(t, uint64(2), dex.GetLastFeeDividendIdFromStorage(db))

	checkFeeSum(t, db, 1, 0, true, 2)
	checkFeeSum(t, db, 2, 1, true, 2)
	feeSum3, _ := dex.GetFeeSumByPeriodIdFromStorage(db, 3)
	assert.True(t, feeSum3 == nil)
	checkAccount(t, db, userAddress0, ETH.tokenId, 70)
	checkAccount(t, db, userAddress1, ETH.tokenId, 50)
	checkAccount(t, db, userAddress0, VITE.tokenId, 52)
	checkAccount(t, db, userAddress1, VITE.tokenId, 38)
	checkVxSumLen(t, db, 3)
	checkUserVxLen(t, db, userAddress0.Bytes(), 2)
	checkUserVxLen(t, db, userAddress1.Bytes(), 3)

	err = feeDividend(db, 4)
	invalidPeriodIdErr := dex.ErrEvent{}
	assert.Equal(t, invalidPeriodIdErr.GetTopicId().Bytes(), db.logList[0].Topics[0].Bytes())
	assert.Equal(t, "fee dividend period id not equals to expected id 3", invalidPeriodIdErr.FromBytes(db.logList[0].Data).(dex.ErrEvent).Error())

	err = feeDividend(db, 3) // not fount feeSum
	err = feeDividend(db, 4) // vxSum is zero
	err = feeDividend(db, 5) // not fount vxSum
	vxSumFund := checkVxSumLen(t, db, 3)
	assert.Equal(t, uint64(6), vxSumFund.Funds[2].Period)
	checkUserVxLen(t, db, userAddress0.Bytes(), 2)
	checkUserVxLen(t, db, userAddress1.Bytes(), 3)
	checkFeeSum(t, db, 4, 2, false, 2)

	err = feeDividend(db, 6)
	// vxSumFunds 6 -> 21
	// userAddress0 vxFund nil or (3 -> 20, 4 -> 18)
	// userAddress1 vxFund 6 -> 21
	// feeSum 4 -> [ETH : 19, VITE : 30]
	// userAddress0 userFees 4 -> [VITE : 13]
	// userAddress1 userFees 4 -> [VITE : 17]
	// userAddress2 userFees 4 -> [ETH : 19]
	checkVxSumLen(t, db, 1)
	checkFeeSum(t, db, 4, 2, true, 2)
	// if userAddress0 deal first it will be nil, else it will not be changed
	vxFunds, _ := dex.GetVxFundsFromStorage(db, userAddress0.Bytes())
	assert.True(t, vxFunds == nil || len(vxFunds.Funds) == 2)

	checkUserVxLen(t, db, userAddress1.Bytes(), 1)
	checkAccount(t, db, userAddress1, ETH.tokenId, 69) // 50 + 19
	checkAccount(t, db, userAddress1, VITE.tokenId, 68) // 38 + 13 + 17
}

func innerTestMinedVxForFee(t *testing.T, db *testDatabase) {
	vxMinedVxDividend(db, 1)
}

func checkVxSumLen(t *testing.T, db *testDatabase, expectedLen int) *dex.VxFunds {
	vxSumFunds, _ := dex.GetVxSumFundsFromDb(db)
	assert.Equal(t, expectedLen, len(vxSumFunds.Funds))
	return vxSumFunds
}

func checkUserVxLen(t *testing.T, db *testDatabase, address []byte, expectedLen int) *dex.VxFunds {
	vxFunds, _ := dex.GetVxFundsFromStorage(db, address)
	assert.Equal(t, expectedLen, len(vxFunds.Funds))
	return vxFunds
}

func checkFeeSum(t *testing.T, db *testDatabase, periodId, expectedLastPeriod uint64, feeDividedStatus bool, feeLen int) {
	feeSum1, _ := dex.GetFeeSumByPeriodIdFromStorage(db, periodId)
	assert.Equal(t, expectedLastPeriod, feeSum1.LastValidPeriod)
	assert.True(t, feeSum1.FeeDivided == feeDividedStatus)
	assert.Equal(t, feeLen, len(feeSum1.Fees))
}

func checkAccount(t *testing.T, db *testDatabase, address types.Address, tokenId types.TokenTypeId, amount int) {
	userFund, _ := dex.GetUserFundFromStorage(db, address)
	account, _ := dex.GetAccountByTokeIdFromFund(userFund, tokenId)
	assert.True(t, CheckBigEqualToInt(amount, account.Available))
}

func deposit(db *testDatabase, address types.Address, tokenId types.TokenTypeId, amount int64) error {
	var err error
	depositSendBlock := &ledger.AccountBlock{}
	depositSendBlock.AccountAddress = address

	depositSendBlock.TokenId = tokenId
	depositSendBlock.Amount = big.NewInt(amount)
	depositSendBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserDeposit)
	depositReceiveBlock := &ledger.AccountBlock{}

	depositMethod := contracts.MethodDexFundUserDeposit{}
	err = doDeposit(db, depositMethod, depositReceiveBlock, depositSendBlock)
	return err
}

func withdraw(db *testDatabase, address types.Address, tokenId types.TokenTypeId, amount int64) error {
	var err error
	withdrawSendBlock := &ledger.AccountBlock{}
	withdrawSendBlock.AccountAddress = address

	withdrawSendBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundUserWithdraw, tokenId, big.NewInt(amount))
	withdrawReceiveBlock := &ledger.AccountBlock{}

	withdrawMethod := contracts.MethodDexFundUserWithdraw{}
	_, err = doWithdraw(db, withdrawMethod, withdrawReceiveBlock, withdrawSendBlock)
	return err
}

func settleFund(db *testDatabase, address types.Address, tokenId types.TokenTypeId, incAmount int64) error {
	addBalance(db, tokenId, big.NewInt(incAmount))
	return settleFundAndFee(db, address, tokenId, incAmount, vxTokenId, 0)
}

func settleFee(db *testDatabase, address types.Address, feeTokenId types.TokenTypeId, feeAmount int64) error {
	addBalance(db, feeTokenId, big.NewInt(feeAmount))
	return settleFundAndFee(db, address, vxTokenId, 0, feeTokenId, feeAmount)
}
func settleFundAndFee(db *testDatabase, address types.Address, tokenId types.TokenTypeId, incAmount int64, feeTokenId types.TokenTypeId, feeAmount int64) error {
	settleActions := &dexproto.SettleActions{}
	if incAmount > 0 {
		fundSettle := &dexproto.FundSettle{}
		fundSettle.Token = tokenId.Bytes()
		fundSettle.IncAvailable = big.NewInt(incAmount).Bytes()

		fundAction := &dexproto.UserFundSettle{}
		fundAction.Address = address.Bytes()
		fundAction.FundSettles = append(fundAction.FundSettles, fundSettle)
		settleActions.FundActions = append(settleActions.FundActions, fundAction)
	}

	if feeAmount > 0 {
		userFeeSettle := &dexproto.UserFeeSettle{}
		userFeeSettle.Address = address.Bytes()
		userFeeSettle.Amount = big.NewInt(feeAmount).Bytes()

		feeAction := &dexproto.FeeSettle{}
		feeAction.Token = feeTokenId.Bytes()
		feeAction.UserFeeSettles = append(feeAction.UserFeeSettles, userFeeSettle)
		settleActions.FeeActions = append(settleActions.FeeActions, feeAction)
	}

	if data, err := proto.Marshal(settleActions); err !=nil {
		return err
	} else {
		settleSendBlock := &ledger.AccountBlock{}
		settleSendBlock.AccountAddress = types.AddressDexTrade
		if settleSendBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundSettleOrders, data); err != nil {
			return err
		} else {
			settleReceiveBlock := &ledger.AccountBlock{}
			settleMethod := contracts.MethodDexFundSettleOrders{}
			_, err = settleMethod.DoReceive(db, settleReceiveBlock, settleSendBlock, nil)
			return err
		}
	}
}

func feeDividend(db *testDatabase, periodId uint64) error {
	var err error
	feeDividendSendBlock := &ledger.AccountBlock{}
	if feeDividendSendBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundFeeDividend, periodId); err != nil {
		return  err
	} else {
		feeDividendReceiveBlock := &ledger.AccountBlock{}
		feeDividendMethod := contracts.MethodDexFundFeeDividend{}
		_, err = feeDividendMethod.DoReceive(db, feeDividendReceiveBlock, feeDividendSendBlock, nil)
		return err
	}
}

func vxMinedVxDividend(db *testDatabase, periodId uint64) error {
	var err error
	vxMinedVxDividendSendBlock := &ledger.AccountBlock{}
	if vxMinedVxDividendSendBlock.Data, err = contracts.ABIDexFund.PackMethod(contracts.MethodNameDexFundMinedVxDividend, periodId); err != nil {
		return  err
	} else {
		vxMinedVxDividendReceiveBlock := &ledger.AccountBlock{}
		minedVxDividendMethod := contracts.MethodDexFundMinedVxDividend{}
		_, err = minedVxDividendMethod.DoReceive(db, vxMinedVxDividendReceiveBlock, vxMinedVxDividendSendBlock, nil)
		return err
	}
}

func getConsensusReader() util.ConsensusReader {
	return nil
}

func addBalance(db *testDatabase, tokenId types.TokenTypeId, amount *big.Int) {
	origin, _ := db.GetBalance(&tokenId)
	db.SetBalance(&tokenId, origin.Add(origin, amount))
}

func subBalance(db *testDatabase, tokenId types.TokenTypeId, amount *big.Int) {
	origin, _ := db.GetBalance(&tokenId)
	db.SetBalance(&tokenId, origin.Sub(origin, amount))
}