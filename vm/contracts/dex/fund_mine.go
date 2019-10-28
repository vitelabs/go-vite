package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

//Note: allow mine from specify periodId, former periods will be ignore
func DoMineVxForFee(db vm_db.VmDb, reader util.ConsensusReader, periodId uint64, amtForMarkets map[int32]*big.Int, fundLogger log15.Logger) (*big.Int, error) {
	var (
		dexFeesByPeriod                *DexFeesByPeriod
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
	if dexFeesByPeriod, ok = GetDexFeesByPeriodId(db, periodId); !ok {
		return AccumulateAmountFromMap(amtForMarkets), nil
	}
	for _, feeForMine := range dexFeesByPeriod.FeesForMine {
		feeSumMap[feeForMine.QuoteTokenType] = new(big.Int).SetBytes(AddBigInt(feeForMine.BaseAmount, feeForMine.InviteBonusAmount))
		dividedFeeMap[feeForMine.QuoteTokenType] = big.NewInt(0)
	}
	for i := ViteTokenType; i <= UsdTokenType; i++ {
		mineThresholdMap[int32(i)] = GetMineThreshold(db, int32(i))
		toDivideVxLeaveAmtMap[int32(i)] = new(big.Int).Set(amtForMarkets[int32(i)])
	}

	MarkDexFeesFinishMine(db, dexFeesByPeriod, periodId)
	var (
		userFeesKey, userFeesBytes []byte
	)

	iterator, err := db.NewStorageIterator(userFeeKeyPrefix)
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
		if len(userFees.Fees[0].Fees) > 0 {
			var vxMinedForBase = big.NewInt(0)
			var vxMinedForInvite = big.NewInt(0)
			for _, feeAccount := range userFees.Fees[0].Fees {
				if !IsValidFeeForMine(feeAccount, mineThresholdMap[feeAccount.QuoteTokenType]) {
					continue
				}
				if feeSumAmt, ok := feeSumMap[feeAccount.QuoteTokenType]; !ok { //no counter part in feeSum for userFees
					// TODO change to continue after test
					fundLogger.Error("DoMineVxForFee", "encounter err" , "user with valid feeAccount, but no valid feeSum",
						"periodId", periodId, "address", address.String(), "quoteTokenType", feeAccount.QuoteTokenType,
						"baseFee", new(big.Int).SetBytes(feeAccount.BaseAmount), "inviteFee", new(big.Int).SetBytes(feeAccount.InviteBonusAmount))
					continue
				} else {
					var vxDividend, vxDividendForInvite *big.Int
					var finished, finishedForInvite bool
					if len(feeAccount.BaseAmount) > 0 {
						vxDividend, finished = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(feeAccount.BaseAmount), dividedFeeMap[feeAccount.QuoteTokenType], amtForMarkets[feeAccount.QuoteTokenType], toDivideVxLeaveAmtMap[feeAccount.QuoteTokenType])
						vxMinedForBase.Add(vxMinedForBase, vxDividend)
						AddMinedVxForTradeFeeEvent(db, address, feeAccount.QuoteTokenType, feeAccount.BaseAmount, vxDividend)
					}
					if finished {
						delete(feeSumMap, feeAccount.QuoteTokenType)
					} else {
						if len(feeAccount.InviteBonusAmount) > 0 {
							vxDividendForInvite, finishedForInvite = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(feeAccount.InviteBonusAmount), dividedFeeMap[feeAccount.QuoteTokenType], amtForMarkets[feeAccount.QuoteTokenType], toDivideVxLeaveAmtMap[feeAccount.QuoteTokenType])
							vxMinedForInvite.Add(vxMinedForInvite, vxDividendForInvite)
							AddMinedVxForInviteeFeeEvent(db, address, feeAccount.QuoteTokenType, feeAccount.InviteBonusAmount, vxDividendForInvite)
							if finishedForInvite {
								delete(feeSumMap, feeAccount.QuoteTokenType)
							}
						}
					}
				}
			}
			minedAmt := new(big.Int).Add(vxMinedForBase, vxMinedForInvite)
			if minedAmt.Sign() > 0 {
				updatedAcc := DepositAccount(db, address, VxTokenId, minedAmt)
				if err = OnVxMined(db, reader, address, minedAmt, updatedAcc); err != nil {
					return nil, err
				}
			}
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

func DoMineVxForStaking(db vm_db.VmDb, reader util.ConsensusReader, periodId uint64, amountToMine *big.Int) (*big.Int, error) {
	var (
		dexMiningStakings      *MiningStakings
		dividedStakedAmountSum = big.NewInt(0)
		amtLeavedToMine        = new(big.Int).Set(amountToMine)
		ok                     bool
	)
	if amountToMine == nil {
		return nil, nil
	}
	if dexMiningStakings, ok = GetDexMiningStakings(db); !ok {
		return amountToMine, nil
	}
	foundDexMiningStaking, dexMiningStakedAmountBytes, needUpdateDexMiningStakings, _ := MatchMiningStakingByPeriod(dexMiningStakings, periodId, false)
	if !foundDexMiningStaking { // not found vxSumFunds
		return amountToMine, nil
	}
	if needUpdateDexMiningStakings {
		SaveDexMiningStakings(db, dexMiningStakings)
	}
	dexMiningStakedAmount := new(big.Int).SetBytes(dexMiningStakedAmountBytes)
	if dexMiningStakedAmount.Sign() <= 0 {
		return amountToMine, nil
	}

	var (
		miningStakingsKey, miningStakingsValue []byte
	)

	iterator, err := db.NewStorageIterator(miningStakingsKeyPrefix)
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
		miningStakingsKey = iterator.Key()
		miningStakingsValue = iterator.Value()
		if len(miningStakingsValue) == 0 {
			continue
		}
		addressBytes := miningStakingsKey[len(miningStakingsKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			panic(err)
		}
		miningStakings := &MiningStakings{}
		if err = miningStakings.DeSerialize(miningStakingsValue); err != nil {
			panic(err)
		}
		foundMiningStaking, miningStakedAmountBytes, needUpdateMiningStakings, needDeleteMiningStakings := MatchMiningStakingByPeriod(miningStakings, periodId, true)
		if !foundMiningStaking {
			continue
		}
		if needDeleteMiningStakings {
			DeleteMiningStakings(db, address)
		} else if needUpdateMiningStakings {
			SaveMiningStakings(db, address, miningStakings)
		}
		stakedAmt := new(big.Int).SetBytes(miningStakedAmountBytes)
		if !IsValidMiningStakeAmount(stakedAmt) {
			continue
		}
		//fmt.Printf("tokenId %s, address %s, vxSumAmt %s, userVxAmount %s, dividedVxAmt %s, toDivideFeeAmt %s, toDivideLeaveAmt %s\n", tokenId.String(), address.String(), vxSumAmt.String(), userVxAmount.String(), dividedVxAmtMap[tokenId], toDivideFeeAmt.String(), toDivideLeaveAmt.String())
		minedAmt, finished := DivideByProportion(dexMiningStakedAmount, stakedAmt, dividedStakedAmountSum, amountToMine, amtLeavedToMine)
		if minedAmt.Sign() > 0 {
			updatedAcc := DepositAccount(db, address, VxTokenId, minedAmt)
			if err = OnVxMined(db, reader, address, minedAmt, updatedAcc); err != nil {
				return amtLeavedToMine, err
			}
			AddMinedVxForStakingEvent(db, address, stakedAmt, minedAmt)
		}
		if finished {
			break
		}
	}
	return amtLeavedToMine, nil
}

func DoMineVxForMakerMineAndMaintainer(db vm_db.VmDb, periodId uint64, reader util.ConsensusReader, amtForMakerAndMaintainer map[int32]*big.Int) error {
	if amtForMakerAndMaintainer[MineForMaker].Sign() > 0 {
		makerMineProxy := GetMakerMiningAdmin(db)
		amtForMaker, _ := amtForMakerAndMaintainer[MineForMaker]
		SaveMakerMiningPoolByPeriodId(db, periodId, amtForMaker)
		AddMinedVxForOperationEvent(db, MineForMaker, *makerMineProxy, amtForMaker)
	}
	if amtForMakerAndMaintainer[MineForMaintainer].Sign() > 0 {
		maintainer := GetMaintainer(db)
		amtForMaintainer, _ := amtForMakerAndMaintainer[MineForMaintainer]
		updatedAcc := DepositAccount(db, *maintainer, VxTokenId, amtForMaintainer)
		if err := OnVxMined(db, reader, *maintainer, amtForMaintainer, updatedAcc); err != nil {
			return err
		}
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
	if !IsNormalMineStarted(db) {
		return PreheatMinedAmtPerPeriod
	} else {
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
