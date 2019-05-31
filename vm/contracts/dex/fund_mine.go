package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func DoDivideMinedVxForFee(db vm_db.VmDb, periodId uint64, minedVxAmtPerMarket *big.Int) error {
	var (
		feeSum                *FeeSumByPeriod
		feeSumMap             = make(map[types.TokenTypeId]*big.Int)
		dividedFeeMap         = make(map[types.TokenTypeId]*big.Int)
		toDivideVxLeaveAmtMap = make(map[types.TokenTypeId]*big.Int)
		tokenId               types.TokenTypeId
		err                   error
		ok                    bool
	)
	if feeSum, ok = GetFeeSumByPeriodId(db, periodId); !ok {
		return nil
	}
	for _, feeSum := range feeSum.Fees {
		if tokenId, err = types.BytesToTokenTypeId(feeSum.Token); err != nil {
			return err
		}
		feeSumMap[tokenId] = new(big.Int).SetBytes(AddBigInt(feeSum.BaseAmount, feeSum.InviteBonusAmount))
		toDivideVxLeaveAmtMap[tokenId] = minedVxAmtPerMarket
		dividedFeeMap[tokenId] = big.NewInt(0)
	}

	MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)

	var (
		userFeesKey, userFeesBytes []byte
	)

	vxTokenId := types.TokenTypeId{}
	vxTokenId.SetBytes(VxTokenBytes)
	iterator, err := db.NewStorageIterator(UserFeeKeyPrefix)
	if err != nil {
		return err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			break
		} else {
			userFeesKey = iterator.Key()
			userFeesBytes = iterator.Value()
			if len(userFeesBytes) == 0 {
				continue
			}
		}

		addressBytes := userFeesKey[len(UserFeeKeyPrefix):]
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
				if tokenId, err = types.BytesToTokenTypeId(userFee.Token); err != nil {
					return err
				}
				if feeSumAmt, ok := feeSumMap[tokenId]; !ok { //no counter part in feeSum for userFees
					// TODO change to continue after test
					return fmt.Errorf("user with valid userFee, but no valid feeSum")
					//continue
				} else {
					vxDividend, finished := DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.BaseAmount), dividedFeeMap[tokenId], minedVxAmtPerMarket, toDivideVxLeaveAmtMap[tokenId])
					vxMinedForBase.Add(vxMinedForBase, vxDividend)
					if finished {
						delete(feeSumMap, tokenId)
					} else {
						vxDividend, finished = DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.InviteBonusAmount), dividedFeeMap[tokenId], minedVxAmtPerMarket, toDivideVxLeaveAmtMap[tokenId])
						vxMinedForInvite.Add(vxMinedForInvite, vxDividend)
						if finished {
							delete(feeSumMap, tokenId)
						}
					}
				}
			}
			if err = BatchSaveUserFund(db, address, map[types.TokenTypeId]*big.Int{vxTokenId: new(big.Int).Add(vxMinedForBase, vxMinedForInvite)}); err != nil {
				return err
			}
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

func DoDivideMinedVxForPledge(db vm_db.VmDb, minedVxAmt *big.Int) error {
	// support accumulate history pledge vx
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	return nil
}

func DoDivideMinedVxForMaintainer(db vm_db.VmDb, minedVxAmt *big.Int) error {
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	if owner, err := GetOwner(db); owner == nil || err != nil {
		panic(InternalErr)
	} else {
		BatchSaveUserFund(db, *owner, map[types.TokenTypeId]*big.Int{VxToken: minedVxAmt})
	}
	return nil
}

func GetMindedVxAmt(db vm_db.VmDb, periodId uint64, vxBalance *big.Int) (amtForFeePerMarket, amtForPledge, amtForMaintainer, vxAmtLeaved *big.Int, success bool) {
	if vxBalance.Sign() > 0 {
		toDivideTotal := GetVxToMineByPeriodId(db, periodId)
		if vxBalance.Cmp(toDivideTotal) < 0 {
			toDivideTotal = vxBalance
		}
		toDivideTotalF := new(big.Float).SetPrec(bigFloatPrec).SetInt(toDivideTotal)
		proportion, _ := new(big.Float).SetPrec(bigFloatPrec).SetString("0.15")
		amtForFeePerMarket = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		amtForFeeTotal := new(big.Int).Mul(amtForFeePerMarket, big.NewInt(4))
		proportion, _ = new(big.Float).SetPrec(bigFloatPrec).SetString("0.1")
		amtForMaintainer = RoundAmount(new(big.Float).SetPrec(bigFloatPrec).Mul(toDivideTotalF, proportion))
		amtForPledge = new(big.Int).Sub(toDivideTotal, amtForFeeTotal)
		amtForPledge.Sub(amtForPledge, amtForMaintainer)
		//TODO complete
		return amtForFeePerMarket, amtForPledge, amtForMaintainer, new(big.Int).Sub(vxBalance, toDivideTotal), true
	} else {
		return nil, nil, nil, nil, false
	}
}

func GetVxToMineByPeriodId(db vm_db.VmDb, periodId uint64) *big.Int {
	var firstPeriodId uint64
	if firstPeriodId = GetFirstMinedVxPeriodId(db); firstPeriodId == 0 {
		firstPeriodId = periodId
		SaveFirstMinedVxPeriodId(db, firstPeriodId)
	}
	var amount = new(big.Int).Set(VxMinedAmtFirstPeriod)
	for i := 1; firstPeriodId+uint64(i) <= periodId; i++ {
		if i <= 364 {
			amount.Mul(amount, big.NewInt(995)).Div(amount, big.NewInt(1000))
		} else {
			amount.Mul(amount, big.NewInt(998)).Div(amount, big.NewInt(1000))
		}
	}
	return amount
}
