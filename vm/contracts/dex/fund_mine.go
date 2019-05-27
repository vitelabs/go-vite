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
		feeSumMap[tokenId] = new(big.Int).SetBytes(AddBigInt(feeSum.BaseAmount, feeSum.BrokerAmount))
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
					vxDividend, finished := DivideByProportion(feeSumAmt, new(big.Int).SetBytes(AddBigInt(userFee.BaseAmount, userFee.BrokerAmount)), dividedFeeMap[tokenId], minedVxAmtPerMarket, toDivideVxLeaveAmtMap[tokenId])
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

func DoDivideMinedVxForViteXMaintainer(db vm_db.VmDb, minedVxAmt *big.Int) error {
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	return nil
}