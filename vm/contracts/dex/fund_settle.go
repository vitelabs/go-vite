package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func DoSettleFund(db vm_db.VmDb, reader util.ConsensusReader, action *dexproto.UserFundSettle, marketInfo *MarketInfo) error {
	address := types.Address{}
	address.SetBytes([]byte(action.Address))
	dexFund, _ := GetUserFund(db, address)
	for _, fundSettle := range action.FundSettles {
		var token []byte
		if fundSettle.IsTradeToken {
			token = marketInfo.TradeToken
		} else {
			token = marketInfo.QuoteToken
		}
		if tokenId, err := types.BytesToTokenTypeId(token); err != nil {
			return err
		} else {
			if _, ok := GetTokenInfo(db, tokenId); !ok {
				panic(InvalidTokenErr)
			}
			account, exists := GetAccountByTokeIdFromFund(dexFund, tokenId)
			//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
			if CmpToBigZero(fundSettle.ReduceLocked) != 0 {
				if CmpForBigInt(fundSettle.ReduceLocked, account.Locked) > 0 {
					panic(ExceedFundLockedErr)
				}
				account.Locked = SubBigIntAbs(account.Locked, fundSettle.ReduceLocked)
			}
			if CmpToBigZero(fundSettle.ReleaseLocked) != 0 {
				if CmpForBigInt(fundSettle.ReleaseLocked, account.Locked) > 0 {
					panic(ExceedFundLockedErr)
				}
				account.Locked = SubBigIntAbs(account.Locked, fundSettle.ReleaseLocked)
				account.Available = AddBigInt(account.Available, fundSettle.ReleaseLocked)
			}
			if CmpToBigZero(fundSettle.IncAvailable) != 0 {
				account.Available = AddBigInt(account.Available, fundSettle.IncAvailable)
			}
			if !exists {
				dexFund.Accounts = append(dexFund.Accounts, account)
			}
			// must do after account updated by settle
			if bytes.Equal(token, VxTokenBytes) {
				OnSettleVx(db, reader, action.Address, fundSettle, account)
			}
			//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
		}
	}
	SaveUserFund(db, address, dexFund)
	return nil
}

//include
func SettleFeeSum(db vm_db.VmDb, reader util.ConsensusReader, feeActions []*dexproto.UserFeeSettle, feeForDividend *big.Int, feeToken []byte) {
	feeSumByPeriod, ok := GetCurrentFeeSum(db, reader)
	if !ok { // need roll period when current period feeSum not saved yet
		feeSumByPeriod = rollFeeSum(db, reader)
	}
	if len(feeActions) > 0 {
		feeAmountMap := make(map[types.TokenTypeId][]byte)
		brokerFeeAmountMap := make(map[types.TokenTypeId][]byte)
		tokenId, _ := types.BytesToTokenTypeId(feeToken)
		for _, feeAction := range feeActions {
			feeAmountMap[tokenId] = AddBigInt(feeAmountMap[tokenId], feeAction.BaseFee)
			brokerFeeAmountMap[tokenId] = AddBigInt(brokerFeeAmountMap[tokenId], feeAction.BrokerFee)
		}

		for _, feeAcc := range feeSumByPeriod.Fees {
			tokenId, _ := types.BytesToTokenTypeId(feeAcc.Token)
			if incFeeAmount, ok := feeAmountMap[tokenId]; ok {
				feeAcc.BaseAmount = AddBigInt(feeAcc.BaseAmount, incFeeAmount)
				feeAcc.BrokerAmount = AddBigInt(feeAcc.BrokerAmount, brokerFeeAmountMap[tokenId])
				delete(feeAmountMap, tokenId)
				delete(brokerFeeAmountMap, tokenId)
			}
		}
		for tokenId, feeAmount := range feeAmountMap {
			feeSumByPeriod.Fees = append(feeSumByPeriod.Fees, newFeeSumAccount(tokenId.Bytes(), feeAmount, brokerFeeAmountMap[tokenId], nil))
		}
	}
	if feeForDividend != nil {
		var foundVite bool
		for _, feeAcc := range feeSumByPeriod.Fees {
			if bytes.Equal(feeAcc.Token, ledger.ViteTokenId.Bytes()) {
				feeAcc.DividendPoolAmount = AddBigInt(feeAcc.DividendPoolAmount, feeForDividend.Bytes())
				foundVite = true
			}
		}
		if !foundVite {
			feeSumByPeriod.Fees = append(feeSumByPeriod.Fees, newFeeSumAccount(ledger.ViteTokenId.Bytes(), nil, nil, feeForDividend.Bytes()))
		}
	}
	SaveCurrentFeeSum(db, reader, feeSumByPeriod)
}

//baseAmount + brokerAmount for vx mine,
func SettleUserFees(db vm_db.VmDb, reader util.ConsensusReader, feeAction *dexproto.UserFeeSettle, feeToken []byte) {
	periodId := GetCurrentPeriodId(db, reader)
	userFees, _ := GetUserFees(db, feeAction.Address)
	feeLen := len(userFees.Fees)
	if feeLen > 0 && periodId == userFees.Fees[feeLen-1].Period {
		var foundToken = false
		for _, feeAcc := range userFees.Fees[feeLen-1].UserFees {
			if bytes.Equal(feeAcc.Token, feeToken) {
				feeAcc.BaseAmount = AddBigInt(feeAcc.BaseAmount, feeAction.BaseFee)
				feeAcc.BrokerAmount = AddBigInt(feeAcc.BrokerAmount, feeAction.BrokerFee)
				foundToken = true
				break
			}
		}
		if !foundToken {
			userFees.Fees[feeLen-1].UserFees = append(userFees.Fees[feeLen-1].UserFees, newFeeAccount(feeToken, feeAction.BaseFee, feeAction.BrokerFee))
		}
	} else {
		userFeeByPeriodId := &dexproto.UserFeeWithPeriod{}
		userFeeByPeriodId.Period = periodId
		userFeeByPeriodId.UserFees = []*dexproto.FeeAccount{newFeeAccount(feeToken, feeAction.BaseFee, feeAction.BrokerFee)}
		userFees.Fees = append(userFees.Fees, userFeeByPeriodId)
	}
	SaveUserFees(db, feeAction.Address, userFees)
}

func SettleBrokerFeeSum(db vm_db.VmDb, reader util.ConsensusReader, feeActions []*dexproto.UserFeeSettle, marketInfo *MarketInfo) {
	var incAmt []byte
	for _, feeAction := range feeActions {
		incAmt = AddBigInt(incAmt, feeAction.BrokerFee)
	}

	var feeToken = marketInfo.QuoteToken
	var settleBrokerAddress = marketInfo.Owner
	var foundBroker bool
	brokerFeeSumByPeriod, _ := GetCurrentBrokerFeeSum(db, reader)
	for _, brokerFeeSum := range brokerFeeSumByPeriod.BrokerFees {
		if bytes.Equal(settleBrokerAddress, brokerFeeSum.Broker) {
			var found bool
			for _, feeAcc := range brokerFeeSum.Fees {
				if bytes.Equal(feeAcc.Token, feeToken) {
					feeAcc.BrokerAmount = AddBigInt(feeAcc.BrokerAmount, incAmt)
					found = true
				}
			}
			if !found {
				brokerFeeSum.Fees = append(brokerFeeSum.Fees, newFeeAccount(feeToken, nil, incAmt))
			}
			foundBroker = true
		}
	}
	if !foundBroker {
		brokerFeeSum := &dexproto.BrokerFeeSum{}
		brokerFeeSum.Broker = settleBrokerAddress
		brokerFeeSum.Fees = append(brokerFeeSum.Fees, newFeeAccount(feeToken, nil, incAmt))
		brokerFeeSumByPeriod.BrokerFees = append(brokerFeeSumByPeriod.BrokerFees, brokerFeeSum)
	}
	SaveCurrentBrokerFeeSum(db, reader, brokerFeeSumByPeriod)
}

func OnDepositVx(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, depositAmount *big.Int, updatedVxAccount *dexproto.Account) {
	doSettleVxFunds(db, reader, address.Bytes(), depositAmount, updatedVxAccount)
}

func OnWithdrawVx(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, withdrawAmount *big.Int, updatedVxAccount *dexproto.Account) {
	doSettleVxFunds(db, reader, address.Bytes(), new(big.Int).Neg(withdrawAmount), updatedVxAccount)
}

func OnSettleVx(db vm_db.VmDb, reader util.ConsensusReader, address []byte, fundSettle *dexproto.FundSettle, updatedVxAccount *dexproto.Account) {
	amtChange := SubBigInt(fundSettle.IncAvailable, fundSettle.ReduceLocked)
	doSettleVxFunds(db, reader, address, amtChange, updatedVxAccount)
}

func rollFeeSum(db vm_db.VmDb, reader util.ConsensusReader) (rolledFeeSumByPeriod *FeeSumByPeriod) {
	formerId := GetFeeSumLastPeriodIdForRoll(db)
	rolledFeeSumByPeriod = &FeeSumByPeriod{}
	if formerId > 0 {
		if formerFeeSumByPeriod, ok := GetFeeSumByPeriodId(db, formerId); !ok {
			panic(NoFeeSumFoundForValidPeriodErr)
		} else {
			rolledFeeSumByPeriod.LastValidPeriod = formerId
			for _, fee := range formerFeeSumByPeriod.Fees {
				rolledFee := &dexproto.FeeSumAccount{}
				rolledFee.Token = fee.Token
				_, rolledAmount := splitDividendPool(fee)
				rolledFee.DividendPoolAmount = rolledAmount.Bytes()
				rolledFeeSumByPeriod.Fees = append(rolledFeeSumByPeriod.Fees, rolledFee)
			}
		}
	}
	SaveFeeSumLastPeriodIdForRoll(db, reader)
	return
}

func splitDividendPool(feeSumAcc *dexproto.FeeSumAccount) (toDividendAmt, rolledAmount *big.Int) {
	dividendPoolTotal := AddBigInt(feeSumAcc.DividendPoolAmount, feeSumAcc.BaseAmount)
	toDividendAmt = new(big.Int).SetBytes(CalculateRawFee(dividendPoolTotal, PerPeriodDividendRate)) // %1
	rolledAmount = new(big.Int).Sub(new(big.Int).SetBytes(dividendPoolTotal), toDividendAmt)         // 99%
	return
}

// only settle validAmount and amount changed from previous period
func doSettleVxFunds(db vm_db.VmDb, reader util.ConsensusReader, addressBytes []byte, amtChange *big.Int, updatedVxAccount *dexproto.Account) {
	var (
		vxFunds               *VxFunds
		userNewAmt, sumChange *big.Int
		periodId              uint64
		fundsLen              int
		needUpdate            bool
	)
	vxFunds, _ = GetVxFundsFrom(db, addressBytes)
	periodId = GetCurrentPeriodId(db, reader)
	fundsLen = len(vxFunds.Funds)
	userNewAmt = new(big.Int).SetBytes(AddBigInt(updatedVxAccount.Available, updatedVxAccount.Locked))
	if fundsLen == 0 { //need append new period
		if IsValidVxAmountForDividend(userNewAmt) {
			fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
			sumChange = userNewAmt
			needUpdate = true
		}
	} else if vxFunds.Funds[fundsLen-1].Period == periodId { //update current period
		if IsValidVxAmountForDividend(userNewAmt) {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = userNewAmt
			}
			vxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
		} else {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = NegativeAmount(vxFunds.Funds[fundsLen-1].Amount)
			}
			if fundsLen > 1 { // in case fundsLen > 1, update last period to diff the condition of current period not changed ever from last saved period
				vxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
			} else { // clear funds in case only current period saved and not valid any more
				vxFunds.Funds = nil
			}
		}
		needUpdate = true
	} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
		if IsValidVxAmountForDividend(userNewAmt) {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = userNewAmt
			}
			fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
			needUpdate = true
		} else {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = NegativeAmount(vxFunds.Funds[fundsLen-1].Amount)
				fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
				vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
				needUpdate = true
			}
		}
	}

	if len(vxFunds.Funds) > 0 && needUpdate {
		SaveVxFunds(db, addressBytes, vxFunds)
	} else if len(vxFunds.Funds) == 0 && fundsLen > 0 {
		DeleteVxFunds(db, addressBytes)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		vxSumFunds, _ := GetVxSumFundsFromDb(db)
		sumFundsLen := len(vxSumFunds.Funds)
		if sumFundsLen == 0 {
			if sumChange.Sign() > 0 {
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Period: periodId, Amount: sumChange.Bytes()})
			} else {
				panic(fmt.Errorf("vxFundSum initiation get negative value"))
			}
		} else {
			sumRes := new(big.Int).Add(new(big.Int).SetBytes(vxSumFunds.Funds[sumFundsLen-1].Amount), sumChange)
			if sumRes.Sign() < 0 {
				panic(fmt.Errorf("vxFundSum updated res get negative value"))
			}
			if vxSumFunds.Funds[sumFundsLen-1].Period == periodId {
				vxSumFunds.Funds[sumFundsLen-1].Amount = sumRes.Bytes()
			} else {
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Amount: sumRes.Bytes(), Period: periodId})
			}
		}
		SaveVxSumFundsToDb(db, vxSumFunds)
	}
}

func newFeeAccount(token, baseAmount, brokerAmount []byte) *dexproto.FeeAccount {
	account := &dexproto.FeeAccount{}
	account.Token = token
	account.BaseAmount = baseAmount
	account.BrokerAmount = brokerAmount
	return account
}

func newFeeSumAccount(token, baseAmount, brokerAmount, dividendPoolAmount []byte) *dexproto.FeeSumAccount {
	account := &dexproto.FeeSumAccount{}
	account.Token = token
	account.BaseAmount = baseAmount
	account.BrokerAmount = brokerAmount
	account.DividendPoolAmount = dividendPoolAmount
	return account
}
