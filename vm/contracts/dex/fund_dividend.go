package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func DoDivideFees(db vm_db.VmDb, periodId uint64) error {
	var (
		feeSumsMap    map[uint64]*FeeSumByPeriod
		brokerFeeSums = make(map[uint64]*BrokerFeeSumByPeriod)
		vxSumFunds    *VxFunds
		err           error
		ok            bool
	)

	//allow divide history fees that not divided yet
	if feeSumsMap, brokerFeeSums = GetNotDividedFeeSumsByPeriodId(db, periodId); len(feeSumsMap) == 0 { // no fee to divide
		return nil
	}
	if vxSumFunds, ok = GetVxSumFundsFromDb(db); !ok {
		return nil
	}
	foundVxSumFunds, vxSumAmtBytes, needUpdateVxSum, _ := MatchVxFundsByPeriod(vxSumFunds, periodId, false)
	//fmt.Printf("foundVxSumFunds %v, vxSumAmtBytes %s, needUpdateVxSum %v with periodId %d\n", foundVxSumFunds, new(big.Int).SetBytes(vxSumAmtBytes).String(), needUpdateVxSum, periodId)
	if !foundVxSumFunds { // not found vxSumFunds
		return nil
	}
	if needUpdateVxSum {
		SaveVxSumFundsToDb(db, vxSumFunds)
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
				toDividendAmt, _ := splitDividendPool(feeAccount)
				if amt, ok := feeSumMap[tokenId]; !ok {
					feeSumMap[tokenId] = toDividendAmt
				} else {
					feeSumMap[tokenId] = amt.Add(amt, toDividendAmt)
				}
			}
		}
		MarkFeeSumAsFeeDivided(db, fee, pId)
	}

	var (
		userVxFundsKey, userVxFundsBytes []byte
	)

	iterator, err := db.NewStorageIterator(VxFundKeyPrefix)
	if err != nil {
		return err
	}
	defer iterator.Release()

	feeSumLeavedMap := make(map[types.TokenTypeId]*big.Int)
	dividedVxAmtMap := make(map[types.TokenTypeId]*big.Int)
	for {
		if len(feeSumMap) == 0 {
			break
		}
		if ok = iterator.Next(); ok {
			userVxFundsKey = iterator.Key()
			userVxFundsBytes = iterator.Value()
			if len(userVxFundsBytes) == 0 {
				continue
			}
		} else {
			break
		}

		addressBytes := userVxFundsKey[len(VxFundKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		userVxFunds := &VxFunds{}
		if err = userVxFunds.DeSerialize(userVxFundsBytes); err != nil {
			return err
		}

		var userFeeDividend = make(map[types.TokenTypeId]*big.Int)
		foundVxFunds, userVxAmtBytes, needUpdateVxFunds, needDeleteVxFunds := MatchVxFundsByPeriod(userVxFunds, periodId, true)
		if !foundVxFunds {
			continue
		}
		if needDeleteVxFunds {
			DeleteVxFunds(db, address.Bytes())
		} else if needUpdateVxFunds {
			SaveVxFunds(db, address.Bytes(), userVxFunds)
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
	return DoDividendBrokerFee(db, brokerFeeSums)
}


func DoDividendBrokerFee(db vm_db.VmDb, brokerFeeSums map[uint64]*BrokerFeeSumByPeriod) error {
	var (
		address types.Address
		tokenId types.TokenTypeId
		err error
	)
	for _, brokerFeeSumByPeriod := range brokerFeeSums {
		for _, brokerFeeSum := range brokerFeeSumByPeriod.BrokerFees {
			if address, err = types.BytesToAddress(brokerFeeSum.Broker); err != nil {
				return err
			}
			funds := make(map[types.TokenTypeId]*big.Int)
			for _, acc := range brokerFeeSum.Fees {
				if tokenId, err = types.BytesToTokenTypeId(acc.Token); err != nil {
					return err
				}
				amt := new(big.Int).SetBytes(acc.BrokerAmount)
				if amt.Sign() > 0 {
					funds[tokenId] = amt
				}
			}
			if len(funds) > 0 {
				if err = BatchSaveUserFund(db, address, funds); err != nil {
					return err
				}
			}
		}
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