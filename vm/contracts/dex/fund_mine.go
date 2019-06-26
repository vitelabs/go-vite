package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func DoMineVxForFee(db vm_db.VmDb, reader util.ConsensusReader, periodId uint64, amtPerMarket *big.Int) error {
	var (
		feeSum                *FeeSumByPeriod
		feeSumMap             = make(map[int32]*big.Int) // quoteTokenType -> amount
		dividedFeeMap         = make(map[int32]*big.Int)
		toDivideVxLeaveAmtMap = make(map[int32]*big.Int)
		mineThesholdMap = make(map[int32]*big.Int)
		err                   error
		ok                    bool
	)
	if amtPerMarket == nil {
		return nil
	}
	if feeSum, ok = GetFeeSumByPeriodId(db, periodId); !ok {
		return nil
	}
	for _, feeSum := range feeSum.FeesForMine {
		feeSumMap[feeSum.QuoteTokenType] = new(big.Int).SetBytes(AddBigInt(feeSum.BaseAmount, feeSum.InviteBonusAmount))
		toDivideVxLeaveAmtMap[feeSum.QuoteTokenType] = amtPerMarket
		dividedFeeMap[feeSum.QuoteTokenType] = big.NewInt(0)
		mineThesholdMap[feeSum.QuoteTokenType] = GetMineThreshold(db, feeSum.QuoteTokenType)
	}

	MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)

	var (
		userFeesKey, userFeesBytes []byte
	)

	iterator, err := db.NewStorageIterator(userFeeKeyPrefix)
	if err != nil {
		return err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			userFeesKey = iterator.Key()
			userFeesBytes = iterator.Value()
			if len(userFeesBytes) == 0 {
				continue
			}
		} else {
			break
		}

		addressBytes := userFeesKey[len(userFeeKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		userFees := &UserFees{}
		if err = userFees.DeSerialize(userFeesBytes); err != nil {
			return err
		}
		if userFees.Fees[0].Period != periodId {
			continue
		}
		if len(userFees.Fees[0].UserFees) > 0 {
			var vxMinedForBase = big.NewInt(0)
			var vxMinedForInvite = big.NewInt(0)
			for _, userFee := range userFees.Fees[0].UserFees {
				if !IsValidFeeForMine(userFee, mineThesholdMap[userFee.QuoteTokenType]) {
					continue
				}
				if feeSumAmt, ok := feeSumMap[userFee.QuoteTokenType]; !ok { //no counter part in feeSum for userFees
					// TODO change to continue after test
					return fmt.Errorf("user with valid userFee, but no valid feeSum")
					//continue
				} else {
					var vxDividend, vxDividendForInvite *big.Int
					var finished, finishedForInvite bool
					if len(userFee.BaseAmount) > 0 {
						vxDividend, finished = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.BaseAmount), dividedFeeMap[userFee.QuoteTokenType], amtPerMarket, toDivideVxLeaveAmtMap[userFee.QuoteTokenType])
						vxMinedForBase.Add(vxMinedForBase, vxDividend)
						AddMinedVxForTradeFeeEvent(db, address, userFee.QuoteTokenType, userFee.BaseAmount, vxDividend)
					}
					if finished {
						delete(feeSumMap, userFee.QuoteTokenType)
					} else {
						if len(userFee.InviteBonusAmount) > 0 {
							vxDividendForInvite, finishedForInvite = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.InviteBonusAmount), dividedFeeMap[userFee.QuoteTokenType], amtPerMarket, toDivideVxLeaveAmtMap[userFee.QuoteTokenType])
							vxMinedForInvite.Add(vxMinedForInvite, vxDividendForInvite)
							AddMinedVxForInviteeFeeEvent(db, address, userFee.QuoteTokenType, userFee.InviteBonusAmount, vxDividendForInvite)
							if finishedForInvite {
								delete(feeSumMap, userFee.QuoteTokenType)
							}
						}
					}
				}
			}
			minedAmt := new(big.Int).Add(vxMinedForBase, vxMinedForInvite)
			updatedAcc := DepositUserAccount(db, address, VxTokenId, minedAmt)
			OnDepositVx(db, reader, address, minedAmt, updatedAcc)
		}
		if len(userFees.Fees) == 1 {
			DeleteUserFees(db, addressBytes)
		} else {
			userFees.Fees = userFees.Fees[1:]
			SaveUserFees(db, addressBytes, userFees)
		}
	}
	return nil
}

func DoMineVxForPledge(db vm_db.VmDb, reader util.ConsensusReader, periodId uint64, amtForPledge *big.Int) error {
	var (
		pledgesForVxSum        *PledgesForVx
		dividedPledgeAmountSum = big.NewInt(0)
		amtLeavedToMine        = new(big.Int).Set(amtForPledge)
		ok                     bool
	)
	if amtForPledge == nil {
		return nil
	}
	if pledgesForVxSum, ok = GetPledgesForVxSum(db); !ok {
		return nil
	}
	foundPledgesForVxSum, pledgeForVxSumAmtBytes, needUpdatePledgesForVxSum, _ := MatchPledgeForVxByPeriod(pledgesForVxSum, periodId, false)
	if !foundPledgesForVxSum { // not found vxSumFunds
		return nil
	}
	if needUpdatePledgesForVxSum {
		SavePledgesForVxSum(db, pledgesForVxSum)
	}
	pledgeForVxSumAmt := new(big.Int).SetBytes(pledgeForVxSumAmtBytes)
	if pledgeForVxSumAmt.Sign() <= 0 {
		return nil
	}

	var (
		pledgesForVxKey, pledgeForVxValue []byte
	)

	iterator, err := db.NewStorageIterator(pledgesForVxKeyPrefix)
	if err != nil {
		return err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			pledgesForVxKey = iterator.Key()
			pledgeForVxValue = iterator.Value()
			if len(pledgeForVxValue) == 0 {
				continue
			}
		} else {
			break
		}
		addressBytes := pledgesForVxKey[len(pledgesForVxKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		pledgesForVx := &PledgesForVx{}
		if err = pledgesForVx.DeSerialize(pledgeForVxValue); err != nil {
			return err
		}
		foundPledgesForVx, PledgesForVxAmtBytes, needUpdatePledgesForVx, needDeletePledgesForVx := MatchPledgeForVxByPeriod(pledgesForVx, periodId, true)
		if !foundPledgesForVx {
			continue
		}
		if needDeletePledgesForVx {
			DeletePledgesForVx(db, address)
		} else if needUpdatePledgesForVx {
			SavePledgesForVx(db, address, pledgesForVx)
		}
		pledgeAmt := new(big.Int).SetBytes(PledgesForVxAmtBytes)
		if !IsValidPledgeAmountForVx(pledgeAmt) {
			continue
		}
		//fmt.Printf("tokenId %s, address %s, vxSumAmt %s, userVxAmount %s, dividedVxAmt %s, toDivideFeeAmt %s, toDivideLeaveAmt %s\n", tokenId.String(), address.String(), vxSumAmt.String(), userVxAmount.String(), dividedVxAmtMap[tokenId], toDivideFeeAmt.String(), toDivideLeaveAmt.String())
		minedAmt, finished := DivideByProportion(pledgeForVxSumAmt, pledgeAmt, dividedPledgeAmountSum, amtForPledge, amtLeavedToMine)
		updatedAcc := DepositUserAccount(db, address, VxTokenId, minedAmt)
		OnDepositVx(db, reader, address, minedAmt, updatedAcc)
		AddMinedVxForPledgeEvent(db, address, pledgeAmt, minedAmt)
		if finished {
			break
		}
	}
	return nil
}

func DoMineVxForMakerMineAndMaintainer(db vm_db.VmDb, reader util.ConsensusReader, makerAmt, maintainerAmt *big.Int) error {
	if makerAmt != nil {
		if makerMineProxy, err := GetMakerMineProxy(db); makerMineProxy == nil || err != nil {
			panic(InternalErr)
		} else {
			updatedAcc := DepositUserAccount(db, *makerMineProxy, VxTokenId, makerAmt)
			OnDepositVx(db, reader, *makerMineProxy, makerAmt, updatedAcc)
		}
	}
	if maintainerAmt != nil {
		if maintainer, err := GetMaintainer(db); maintainer == nil || err != nil {
			panic(InternalErr)
		} else {
			updatedAcc := DepositUserAccount(db, *maintainer, VxTokenId, maintainerAmt)
			OnDepositVx(db, reader, *maintainer, maintainerAmt, updatedAcc)
		}
	}
	return nil
}

func GetVxAmountToMine(db vm_db.VmDb, periodId uint64, vxBalance *big.Int) (amtForFeePerMarket, amtForMaker, amtForMaintainer, amtForPledge, vxAmtLeaved *big.Int, success bool) {
	if vxBalance.Sign() > 0 {
		success = true
		toDivideTotal := GetVxToMineByPeriodId(db, periodId)
		if vxBalance.Cmp(toDivideTotal) < 0 {
			toDivideTotal.Set(vxBalance)
		}
		vxAmtLeaved = new(big.Int).Sub(vxBalance, toDivideTotal)

		toDivideLeaved := new(big.Int).Set(toDivideTotal)
		toDivideTotalF := new(big.Float).SetPrec(bigFloatPrec).SetInt(toDivideTotal)
		proportion, _ := new(big.Float).SetPrec(bigFloatPrec).SetString("0.15") // trade fee mine
		amtForFeePerMarket = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		amtForFeeTotal := new(big.Int).Mul(amtForFeePerMarket, big.NewInt(4))
		toDivideLeaved.Sub(toDivideLeaved, amtForFeeTotal)
		if toDivideLeaved.Sign() <= 0 {
			return
		}
		proportion, _ = new(big.Float).SetPrec(bigFloatPrec).SetString("0.1") // order maker
		amtForMaker = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		toDivideLeaved.Sub(toDivideLeaved, amtForMaker)
		if toDivideLeaved.Sign() <= 0 {
			return
		}
		amtForMaintainer = new(big.Int).Set(amtForMaker)
		toDivideLeaved.Sub(toDivideLeaved, amtForMaintainer)
		if toDivideLeaved.Sign() <= 0 {
			return
		}
		amtForPledge = new(big.Int).Set(toDivideLeaved)
		return
	} else {
		success = false
		return
	}
}

func GetVxToMineByPeriodId(db vm_db.VmDb, periodId uint64) *big.Int {
	var firstPeriodId uint64
	if firstPeriodId = GetFirstMinedVxPeriodId(db); firstPeriodId == 0 {
		firstPeriodId = periodId
		SaveFirstMinedVxPeriodId(db, firstPeriodId)
	}
	var amount *big.Int
	for i := 0; firstPeriodId+uint64(i) <= periodId; i++ {
		if i == 0 {
			amount = new(big.Int).Set(VxMinedAmtFirstPeriod)
		} else if i <= 364 {
			amount.Mul(amount, big.NewInt(995)).Div(amount, big.NewInt(1000))
		} else {
			amount.Mul(amount, big.NewInt(998)).Div(amount, big.NewInt(1000))
		}
	}
	return amount
}
