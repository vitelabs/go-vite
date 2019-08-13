package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

//Note: allow mine from specify periodId, former periods will be ignore
func DoMineVxForFee(db vm_db.VmDb, reader util.ConsensusReader, periodId uint64, amtForMarkets map[int32]*big.Int) (*big.Int, error) {
	var (
		feeSum                *FeeSumByPeriod
		feeSumMap             = make(map[int32]*big.Int) // quoteTokenType -> amount
		dividedFeeMap         = make(map[int32]*big.Int)
		toDivideVxLeaveAmtMap = make(map[int32]*big.Int)
		mineThresholdMap      = make(map[int32]*big.Int)
		err                   error
		ok                    bool
	)
	if len(amtForMarkets) == 0 {
		return nil, nil
	}
	if feeSum, ok = GetFeeSumByPeriodId(db, periodId); !ok {
		return AccumulateAmountFromMap(amtForMarkets), nil
	}
	for _, feeSum := range feeSum.FeesForMine {
		feeSumMap[feeSum.QuoteTokenType] = new(big.Int).SetBytes(AddBigInt(feeSum.BaseAmount, feeSum.InviteBonusAmount))
		dividedFeeMap[feeSum.QuoteTokenType] = big.NewInt(0)
	}
	for i := ViteTokenType; i <= UsdTokenType; i++ {
		mineThresholdMap[int32(i)] = GetMineThreshold(db, int32(i))
		toDivideVxLeaveAmtMap[int32(i)] = new(big.Int).Set(amtForMarkets[int32(i)])
	}

	MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)
	var (
		userFeesKey, userFeesBytes []byte
	)

	iterator, err := db.NewStorageIterator(userFeeKeyPrefix)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				panic(iterator.Error())
			}
			break
		}
		userFeesKey = iterator.Key()
		userFeesBytes = iterator.Value()
		if len(userFeesBytes) == 0 {
			continue
		}

		addressBytes := userFeesKey[len(userFeeKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			panic(err)
		}
		userFees := &UserFees{}
		if err = userFees.DeSerialize(userFeesBytes); err != nil {
			panic(err)
		}

		truncated := TruncateUserFeesToPeriod(userFees, periodId)
		if truncated {
			if len(userFees.Fees) == 0 {
				DeleteUserFees(db, addressBytes)
				continue
			} else if userFees.Fees[0].Period != periodId {
				SaveUserFees(db, addressBytes, userFees)
				continue
			}
		}
		if userFees.Fees[0].Period != periodId {
			continue
		}
		if len(userFees.Fees[0].UserFees) > 0 {
			var vxMinedForBase = big.NewInt(0)
			var vxMinedForInvite = big.NewInt(0)
			for _, userFee := range userFees.Fees[0].UserFees {
				if !IsValidFeeForMine(userFee, mineThresholdMap[userFee.QuoteTokenType]) {
					continue
				}
				if feeSumAmt, ok := feeSumMap[userFee.QuoteTokenType]; !ok { //no counter part in feeSum for userFees
					// TODO change to continue after test
					panic(fmt.Errorf("user with valid userFee, but no valid feeSum"))
					//continue
				} else {
					var vxDividend, vxDividendForInvite *big.Int
					var finished, finishedForInvite bool
					if len(userFee.BaseAmount) > 0 {
						vxDividend, finished = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.BaseAmount), dividedFeeMap[userFee.QuoteTokenType], amtForMarkets[userFee.QuoteTokenType], toDivideVxLeaveAmtMap[userFee.QuoteTokenType])
						vxMinedForBase.Add(vxMinedForBase, vxDividend)
						AddMinedVxForTradeFeeEvent(db, address, userFee.QuoteTokenType, userFee.BaseAmount, vxDividend)
					}
					if finished {
						delete(feeSumMap, userFee.QuoteTokenType)
					} else {
						if len(userFee.InviteBonusAmount) > 0 {
							vxDividendForInvite, finishedForInvite = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.InviteBonusAmount), dividedFeeMap[userFee.QuoteTokenType], amtForMarkets[userFee.QuoteTokenType], toDivideVxLeaveAmtMap[userFee.QuoteTokenType])
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
	return AccumulateAmountFromMap(toDivideVxLeaveAmtMap), nil
}

func DoMineVxForPledge(db vm_db.VmDb, reader util.ConsensusReader, periodId uint64, amtForPledge *big.Int) (*big.Int, error) {
	var (
		pledgesForVxSum        *PledgesForVx
		dividedPledgeAmountSum = big.NewInt(0)
		amtLeavedToMine        = new(big.Int).Set(amtForPledge)
		ok                     bool
	)
	if amtForPledge == nil {
		return nil, nil
	}
	if pledgesForVxSum, ok = GetPledgesForVxSum(db); !ok {
		return amtForPledge, nil
	}
	foundPledgesForVxSum, pledgeForVxSumAmtBytes, needUpdatePledgesForVxSum, _ := MatchPledgeForVxByPeriod(pledgesForVxSum, periodId, false)
	if !foundPledgesForVxSum { // not found vxSumFunds
		return amtForPledge, nil
	}
	if needUpdatePledgesForVxSum {
		SavePledgesForVxSum(db, pledgesForVxSum)
	}
	pledgeForVxSumAmt := new(big.Int).SetBytes(pledgeForVxSumAmtBytes)
	if pledgeForVxSumAmt.Sign() <= 0 {
		return amtForPledge, nil
	}

	var (
		pledgesForVxKey, pledgeForVxValue []byte
	)

	iterator, err := db.NewStorageIterator(pledgesForVxKeyPrefix)
	if err != nil {
		panic(err)
	}
	defer iterator.Release()
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				panic(iterator.Error())
			}
			break
		}
		pledgesForVxKey = iterator.Key()
		pledgeForVxValue = iterator.Value()
		if len(pledgeForVxValue) == 0 {
			continue
		}
		addressBytes := pledgesForVxKey[len(pledgesForVxKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			panic(err)
		}
		pledgesForVx := &PledgesForVx{}
		if err = pledgesForVx.DeSerialize(pledgeForVxValue); err != nil {
			panic(err)
		}
		foundPledgesForVx, pledgesForVxAmtBytes, needUpdatePledgesForVx, needDeletePledgesForVx := MatchPledgeForVxByPeriod(pledgesForVx, periodId, true)
		if !foundPledgesForVx {
			continue
		}
		if needDeletePledgesForVx {
			DeletePledgesForVx(db, address)
		} else if needUpdatePledgesForVx {
			SavePledgesForVx(db, address, pledgesForVx)
		}
		pledgeAmt := new(big.Int).SetBytes(pledgesForVxAmtBytes)
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
	return amtLeavedToMine, nil
}

func DoMineVxForMakerMineAndMaintainer(db vm_db.VmDb, periodId uint64, reader util.ConsensusReader, amtForMakerAndMaintainer map[int32]*big.Int) error {
	if amtForMakerAndMaintainer[MineForMaker].Sign() > 0 {
		makerMineProxy := GetMakerMineProxy(db)
		amtForMaker, _ := amtForMakerAndMaintainer[MineForMaker]
		SaveMakerProxyAmountByPeriodId(db, periodId, amtForMaker)
		AddMinedVxForOperationEvent(db, MineForMaker, *makerMineProxy, amtForMaker)
	}
	if amtForMakerAndMaintainer[MineForMaintainer].Sign() > 0 {
		maintainer := GetMaintainer(db)
		amtForMaintainer, _ := amtForMakerAndMaintainer[MineForMaintainer]
		updatedAcc := DepositUserAccount(db, *maintainer, VxTokenId, amtForMaintainer)
		OnDepositVx(db, reader, *maintainer, amtForMaintainer, updatedAcc)
		AddMinedVxForOperationEvent(db, MineForMaintainer, *maintainer, amtForMaintainer)
	}
	return nil
}

func GetVxAmountsForEqualItems(db vm_db.VmDb, periodId uint64, vxPool *big.Int, rateSum string, begin, end int) (amountForItems map[int32]*big.Int, vxAmtLeaved *big.Int, success bool) {
	if vxPool.Sign() > 0 {
		success = true
		toDivideTotal := GetVxToMineByPeriodId(db, periodId)
		toDivideTotalF := new(big.Float).SetPrec(bigFloatPrec).SetInt(toDivideTotal)
		proportion, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(rateSum)
		amountSum := RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		var notEnough bool
		if amountSum.Cmp(vxPool) > 0 {
			amountSum.Set(vxPool)
			notEnough = true
		}
		amount := new(big.Int).Div(amountSum, big.NewInt(int64(end-begin+1)))
		amountForItems = make(map[int32]*big.Int)
		vxAmtLeaved = new(big.Int).Set(vxPool)
		for i := begin; i <= end; i++ {
			if vxAmtLeaved.Cmp(amount) >= 0 {
				amountForItems[int32(i)] = new(big.Int).Set(amount)
			} else {
				amountForItems[int32(i)] = new(big.Int).Set(vxAmtLeaved)
			}
			vxAmtLeaved.Sub(vxAmtLeaved, amountForItems[int32(i)])
		}
		if notEnough || vxAmtLeaved.Cmp(vxMineDust) <= 0 {
			amountForItems[int32(begin)].Add(amountForItems[int32(begin)], vxAmtLeaved)
			vxAmtLeaved.SetInt64(0)
		}
	}
	return
}

func GetVxAmountToMine(db vm_db.VmDb, periodId uint64, vxPool *big.Int, rate string) (amount, vxAmtLeaved *big.Int, success bool) {
	if vxPool.Sign() > 0 {
		success = true
		toDivideTotal := GetVxToMineByPeriodId(db, periodId)
		toDivideTotalF := new(big.Float).SetPrec(bigFloatPrec).SetInt(toDivideTotal)
		proportion, _ := new(big.Float).SetPrec(bigFloatPrec).SetString(rate)
		amount = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		if amount.Cmp(vxPool) > 0 {
			amount.Set(vxPool)
		}
		vxAmtLeaved = new(big.Int).Sub(vxPool, amount)
		if vxAmtLeaved.Sign() > 0 && vxAmtLeaved.Cmp(vxMineDust) <= 0 {
			amount.Add(amount, vxAmtLeaved)
			vxAmtLeaved.SetInt64(0)
		}
	}
	return
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

func AccumulateAmountFromMap(amountMap map[int32]*big.Int) *big.Int {
	sum := big.NewInt(0)
	for _, amt := range amountMap {
		if amt != nil && amt.Sign() > 0 {
			sum.Add(sum, amt)
		}
	}
	return sum
}
