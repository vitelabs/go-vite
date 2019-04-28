package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func SettleFeeSum(db vm_db.VmDb, feeActions []*dexproto.FeeSettle) error {
	var (
		feeSumByPeriod *FeeSumByPeriod
		err            error
	)
	if feeSumByPeriod, err = GetCurrentFeeSumFromStorage(db); err != nil {
		return err
	} else {
		if feeSumByPeriod == nil { // need roll period when current period feeSum not saved yet
			if formerPeriodId, err := rollFeeSumPeriodId(db); err != nil {
				return err
			} else {
				feeSumByPeriod = &FeeSumByPeriod{}
				feeSumByPeriod.LastValidPeriod = formerPeriodId
			}
		}
		feeAmountMap := make(map[types.TokenTypeId][]byte)
		for _, feeAction := range feeActions {
			tokenId, _ := types.BytesToTokenTypeId(feeAction.Token)
			for _, feeAcc := range feeAction.UserFeeSettles {
				feeAmountMap[tokenId] = AddBigInt(feeAmountMap[tokenId], feeAcc.Amount)
			}
		}

		for _, feeAcc := range feeSumByPeriod.Fees {
			tokenId, _ := types.BytesToTokenTypeId(feeAcc.Token)
			if _, ok := feeAmountMap[tokenId]; ok {
				feeAcc.Amount = AddBigInt(feeAcc.Amount, feeAmountMap[tokenId])
				delete(feeAmountMap, tokenId)
			}
		}
		for tokenId, feeAmount := range feeAmountMap {
			newFeeAcc := &dexproto.FeeAccount{}
			newFeeAcc.Token = tokenId.Bytes()
			newFeeAcc.Amount = feeAmount
			feeSumByPeriod.Fees = append(feeSumByPeriod.Fees, newFeeAcc)
		}
	}
	if err = SaveCurrentFeeSumToStorage(db, feeSumByPeriod); err != nil {
		return err
	} else {
		return nil
	}
}

func SettleUserFees(db vm_db.VmDb, feeAction *dexproto.FeeSettle) error {
	var (
		userFees *UserFees
		periodId uint64
		err      error
	)
	if periodId, err = GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	}
	for _, userFeeSettle := range feeAction.UserFeeSettles {
		if userFees, err = GetUserFeesFromStorage(db, userFeeSettle.Address); err != nil {
			return err
		}
		if userFees == nil || len(userFees.Fees) == 0 {
			userFees = &UserFees{}
		}
		feeLen := len(userFees.Fees)
		if feeLen > 0 && periodId == userFees.Fees[feeLen-1].Period {
			var foundToken = false
			for _, feeAcc := range userFees.Fees[feeLen-1].UserFees {
				if bytes.Compare(feeAcc.Token, feeAction.Token) == 0 {
					feeAcc.Amount = AddBigInt(feeAcc.Amount, userFeeSettle.Amount)
					foundToken = true
					break
				}
			}
			if !foundToken {
				feeAcc := &dexproto.FeeAccount{}
				feeAcc.Token = feeAction.Token
				feeAcc.Amount = userFeeSettle.Amount
				userFees.Fees[feeLen-1].UserFees = append(userFees.Fees[feeLen-1].UserFees, feeAcc)
			}
		} else {
			userFeeByPeriodId := &dexproto.UserFeeWithPeriod{}
			userFeeByPeriodId.Period = periodId
			feeAcc := &dexproto.FeeAccount{}
			feeAcc.Token = feeAction.Token
			feeAcc.Amount = userFeeSettle.Amount
			userFeeByPeriodId.UserFees = []*dexproto.FeeAccount{feeAcc}
			userFees.Fees = append(userFees.Fees, userFeeByPeriodId)
		}
		if err = SaveUserFeesToStorage(db, userFeeSettle.Address, userFees); err != nil {
			return err
		}
	}
	return nil
}

func OnDepositVx(db vm_db.VmDb, address types.Address, depositAmount *big.Int, updatedVxAccount *dexproto.Account) error {
	return doSettleVxFunds(db, address.Bytes(), depositAmount, updatedVxAccount)
}

func OnWithdrawVx(db vm_db.VmDb, address types.Address, withdrawAmount *big.Int, updatedVxAccount *dexproto.Account) error {
	return doSettleVxFunds(db, address.Bytes(), new(big.Int).Neg(withdrawAmount), updatedVxAccount)
}

func OnSettleVx(db vm_db.VmDb, address []byte, fundSettle *dexproto.FundSettle, updatedVxAccount *dexproto.Account) error {
	amtChange := SubBigInt(fundSettle.IncAvailable, fundSettle.ReduceLocked)
	return doSettleVxFunds(db, address, amtChange, updatedVxAccount)
}

func rollFeeSumPeriodId(db vm_db.VmDb) (uint64, error) {
	formerId := GetFeeSumLastPeriodIdForRoll(db)
	if err := SaveFeeSumLastPeriodIdForRoll(db); err != nil {
		return 0, err
	} else {
		return formerId, err
	}
}

// only settle validAmount and amount changed from previous period
func doSettleVxFunds(db vm_db.VmDb, addressBytes []byte, amtChange *big.Int, updatedVxAccount *dexproto.Account) error {
	var (
		vxFunds               *VxFunds
		userNewAmt, sumChange *big.Int
		err                   error
		periodId              uint64
		fundsLen              int
		needUpdate            bool
	)
	if vxFunds, err = GetVxFundsFromStorage(db, addressBytes); err != nil {
		return err
	} else {
		if vxFunds == nil {
			vxFunds = &VxFunds{}
		}
		if periodId, err = GetCurrentPeriodIdFromStorage(db); err != nil {
			return err
		}
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
	}
	if len(vxFunds.Funds) > 0 && needUpdate {
		if err = SaveVxFundsToStorage(db, addressBytes, vxFunds); err != nil {
			return err
		}
	} else if len(vxFunds.Funds) == 0 && fundsLen > 0 {
		DeleteVxFundsFromStorage(db, addressBytes)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		var vxSumFunds *VxFunds
		if vxSumFunds, err = GetVxSumFundsFromStorage(db); err != nil {
			return err
		} else {
			if vxSumFunds == nil {
				vxSumFunds = &VxFunds{}
			}
			sumFundsLen := len(vxSumFunds.Funds)
			if sumFundsLen == 0 {
				if sumChange.Sign() > 0 {
					vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Period: periodId, Amount: sumChange.Bytes()})
				} else {
					return fmt.Errorf("vxFundSum initiation get negative value")
				}
			} else {
				sumRes := new(big.Int).Add(new(big.Int).SetBytes(vxSumFunds.Funds[sumFundsLen-1].Amount), sumChange)
				if sumRes.Sign() < 0 {
					return fmt.Errorf("vxFundSum updated res get negative value")
				}
				if vxSumFunds.Funds[sumFundsLen-1].Period == periodId {
					vxSumFunds.Funds[sumFundsLen-1].Amount = sumRes.Bytes()
				} else {
					vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Amount: sumRes.Bytes(), Period: periodId})
				}
			}
			SaveVxSumFundsToStorage(db, vxSumFunds)
		}
	}
	return nil
}

func DoDivideFees(db vm_db.VmDb, periodId uint64) error {
	var (
		feeSumsMap    map[uint64]*FeeSumByPeriod
		donateFeeSums = make(map[uint64]*big.Int)
		vxSumFunds    *VxFunds
		err           error
	)

	//allow divide history fees that not divided yet
	if feeSumsMap, donateFeeSums, err = GetNotDividedFeeSumsByPeriodIdFromStorage(db, periodId); err != nil {
		return err
	} else if len(feeSumsMap) == 0 || len(feeSumsMap) > 4 { // no fee to divide, or fee types more than 4
		return nil
	}

	if vxSumFunds, err = GetVxSumFundsFromStorage(db); err != nil {
		return err
	} else if vxSumFunds == nil {
		return nil
	}
	foundVxSumFunds, vxSumAmtBytes, needUpdateVxSum, _ := MatchVxFundsByPeriod(vxSumFunds, periodId, false)
	//fmt.Printf("foundVxSumFunds %v, vxSumAmtBytes %s, needUpdateVxSum %v with periodId %d\n", foundVxSumFunds, new(big.Int).SetBytes(vxSumAmtBytes).String(), needUpdateVxSum, periodId)
	if !foundVxSumFunds { // not found vxSumFunds
		return nil
	}
	if needUpdateVxSum {
		if err := SaveVxSumFundsToStorage(db, vxSumFunds); err != nil {
			return err
		}
	}
	vxSumAmt := new(big.Int).SetBytes(vxSumAmtBytes)
	if vxSumAmt.Sign() <= 0 {
		return nil
	}
	// sum fees from multi period not divided
	feeSumMap := make(map[types.TokenTypeId]*big.Int)
	for pId, fee := range feeSumsMap {
		for _, feeAccount := range fee.Fees {
			if tokenId, err := types.BytesToTokenTypeId(feeAccount.Token); err != nil {
				return err
			} else {
				if amt, ok := feeSumMap[tokenId]; !ok {
					feeSumMap[tokenId] = new(big.Int).SetBytes(feeAccount.Amount)
				} else {
					feeSumMap[tokenId] = amt.Add(amt, new(big.Int).SetBytes(feeAccount.Amount))
				}
			}
		}

		MarkFeeSumAsFeeDivided(db, fee, pId)
	}

	for pIdForDonateFee, donateFeeSum := range donateFeeSums {
		feeSumMap[ledger.ViteTokenId] = new(big.Int).Add(feeSumMap[ledger.ViteTokenId], donateFeeSum)
		DeleteDonateFeeSum(db, pIdForDonateFee)
	}

	var (
		userVxFundsKey, userVxFundsBytes []byte
		ok                               bool
	)

	iterator := db.NewStorageIterator(&types.AddressDexFund, VxFundKeyPrefix)

	feeSumLeavedMap := make(map[types.TokenTypeId]*big.Int)
	dividedVxAmtMap := make(map[types.TokenTypeId]*big.Int)
	for {
		if len(feeSumMap) == 0 {
			break
		}
		if userVxFundsKey, userVxFundsBytes, ok = iterator.Next(); !ok {
			break
		}

		addressBytes := userVxFundsKey[len(VxFundKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		userVxFunds := &VxFunds{}
		if userVxFunds, err = userVxFunds.DeSerialize(userVxFundsBytes); err != nil {
			return err
		}

		var userFeeDividend = make(map[types.TokenTypeId]*big.Int)
		foundVxFunds, userVxAmtBytes, needUpdateVxFunds, needDeleteVxFunds := MatchVxFundsByPeriod(userVxFunds, periodId, true)
		if !foundVxFunds {
			continue
		}
		if needDeleteVxFunds {
			DeleteVxFundsFromStorage(db, address.Bytes())
		} else if needUpdateVxFunds {
			if err = SaveVxFundsToStorage(db, address.Bytes(), userVxFunds); err != nil {
				return err
			}
		}
		userVxAmount := new(big.Int).SetBytes(userVxAmtBytes)
		//fmt.Printf("address %s, userVxAmount %s, needDeleteVxFunds %v\n", string(address.Bytes()), userVxAmount.String(), needDeleteVxFunds)
		if !IsValidVxAmountForDividend(userVxAmount) { //skip vxAmount not valid for dividend
			continue
		}

		var finished bool
		for tokenId, feeSumAmount := range feeSumMap {
			if _, ok = feeSumLeavedMap[tokenId]; !ok {
				feeSumLeavedMap[tokenId] = new(big.Int).Set(feeSumAmount)
				dividedVxAmtMap[tokenId] = big.NewInt(0)
			}
			//fmt.Printf("tokenId %s, address %s, vxSumAmt %s, userVxAmount %s, dividedVxAmt %s, toDivideFeeAmt %s, toDivideLeaveAmt %s\n", tokenId.String(), address.String(), vxSumAmt.String(), userVxAmount.String(), dividedVxAmtMap[tokenId], toDivideFeeAmt.String(), toDivideLeaveAmt.String())
			userFeeDividend[tokenId], finished = DivideByProportion(vxSumAmt, userVxAmount, dividedVxAmtMap[tokenId], feeSumAmount, feeSumLeavedMap[tokenId])
			if finished {
				delete(feeSumMap, tokenId)
			}
		}
		if err = BatchSaveUserFund(db, address, userFeeDividend); err != nil {
			return err
		}
	}
	return nil
}

func DoDivideMinedVxForFee(db vm_db.VmDb, periodId uint64, minedVxAmtPerMarket *big.Int) error {
	var (
		feeSum                *FeeSumByPeriod
		feeSumMap             = make(map[types.TokenTypeId]*big.Int)
		dividedFeeMap         = make(map[types.TokenTypeId]*big.Int)
		toDivideVxLeaveAmtMap = make(map[types.TokenTypeId]*big.Int)
		tokenId               types.TokenTypeId
		err                   error
	)
	if feeSum, err = GetFeeSumByPeriodIdFromStorage(db, periodId); err != nil {
		return err
	} else if feeSum == nil { // no fee to divide
		return nil
	}
	for _, feeSum := range feeSum.Fees {
		if tokenId, err = types.BytesToTokenTypeId(feeSum.Token); err != nil {
			return err
		}
		feeSumMap[tokenId] = new(big.Int).SetBytes(feeSum.Amount)
		toDivideVxLeaveAmtMap[tokenId] = minedVxAmtPerMarket
		dividedFeeMap[tokenId] = big.NewInt(0)
	}

	MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)

	var (
		userFeesKey, userFeesBytes []byte
		ok                         bool
	)

	vxTokenId := types.TokenTypeId{}
	vxTokenId.SetBytes(VxTokenBytes)
	iterator := db.NewStorageIterator(&types.AddressDexFund, UserFeeKeyPrefix)
	for {
		if userFeesKey, userFeesBytes, ok = iterator.Next(); !ok {
			break
		}

		addressBytes := userFeesKey[len(UserFeeKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		userFees := &UserFees{}
		if userFees, err = userFees.DeSerialize(userFeesBytes); err != nil {
			return err
		}
		if userFees.Fees[0].Period != periodId {
			continue
		}
		if len(userFees.Fees[0].UserFees) > 0 {
			var userVxDividend = big.NewInt(0)
			for _, userFee := range userFees.Fees[0].UserFees {
				if tokenId, err = types.BytesToTokenTypeId(userFee.Token); err != nil {
					return err
				}
				if feeSumAmt, ok := feeSumMap[tokenId]; !ok { //no counter part in feeSum for userFees
					// TODO change to continue after test
					return fmt.Errorf("user with valid userFee, but no valid feeSum")
					//continue
				} else {
					vxDividend, finished := DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.Amount), dividedFeeMap[tokenId], minedVxAmtPerMarket, toDivideVxLeaveAmtMap[tokenId])
					userVxDividend.Add(userVxDividend, vxDividend)
					if finished {
						delete(feeSumMap, tokenId)
					}
				}
			}
			if err = BatchSaveUserFund(db, address, map[types.TokenTypeId]*big.Int{vxTokenId: userVxDividend}); err != nil {
				return err
			}
		}
		if len(userFees.Fees) == 1 {
			DeleteUserFeesFromStorage(db, addressBytes)
		} else {
			userFees.Fees = userFees.Fees[1:]
			if err = SaveUserFeesToStorage(db, addressBytes, userFees); err != nil {
				return err
			}
		}
	}
	return nil
}

func DoDivideMinedVxForPledge(db vm_db.VmDb, minedVxAmt *big.Int) error {
	// support accumulate history pledge vx
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	return nil
}

func DoDivideMinedVxForViteLabs(db vm_db.VmDb, minedVxAmt *big.Int) error {
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	return nil
}

func DivideByProportion(totalReferAmt, partReferAmt, dividedReferAmt, toDivideTotalAmt, toDivideLeaveAmt *big.Int) (proportionAmt *big.Int, finished bool) {
	dividedReferAmt.Add(dividedReferAmt, partReferAmt)
	proportion := new(big.Float).SetPrec(bigFloatPrec).Quo(new(big.Float).SetPrec(bigFloatPrec).SetInt(partReferAmt), new(big.Float).SetPrec(bigFloatPrec).SetInt(totalReferAmt))
	proportionAmt = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(new(big.Float).SetPrec(bigFloatPrec).SetInt(toDivideTotalAmt), proportion))
	toDivideLeaveNewAmt := new(big.Int).Sub(toDivideLeaveAmt, proportionAmt)
	if toDivideLeaveNewAmt.Sign() <= 0 || dividedReferAmt.Cmp(totalReferAmt) >= 0 {
		proportionAmt.Set(toDivideLeaveAmt)
		finished = true
	} else {
		toDivideLeaveAmt.Set(toDivideLeaveNewAmt)
	}
	return proportionAmt, finished
}
