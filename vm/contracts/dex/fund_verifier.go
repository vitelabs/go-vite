package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type FundVerifyRes struct {
	UserCount      int                                   `json:"userCount"`
	BalanceMatched bool                                  `json:"balanceMatched"`
	VerifyItems    map[types.TokenTypeId]*FundVerifyItem `json:"balances"`
}

type FundVerifyItem struct {
	TokenId        types.TokenTypeId `json:"tokenId"`
	Balance        string            `json:"balance"`
	Amount         string            `json:"amount"`
	UserAmount     string            `json:"userAmount"`
	FeeAmount      string            `json:"feeAmount"`
	FeeOccupy      string            `json:"feeOccupy"`
	BalanceMatched bool              `json:"balanceMatched"`
}

func VerifyDexFundBalance(db vm_db.VmDb, reader *util.VMConsensusReader) *FundVerifyRes {
	userAmountMap := make(map[types.TokenTypeId]*big.Int)
	feeAmountMap := make(map[types.TokenTypeId]*big.Int)
	vxAmount := big.NewInt(0)
	verifyItems := make(map[types.TokenTypeId]*FundVerifyItem)
	count, _ := accumulateUserAccount(db, userAmountMap)
	allBalanceMatched := true
	accumulateFeeDividendPool(db, reader, feeAmountMap)
	accumulateBrokerFeeAccount(db, feeAmountMap)
	accumulateVx(db, vxAmount)
	var (
		foundVx, ok, balanceMatched bool
		balance                     *big.Int
	)
	for tokenId, userAmount := range userAmountMap {
		var (
			amount    *big.Int
			feeAmount *big.Int
			feeOccupy string
		)
		if feeAmount, ok = feeAmountMap[tokenId]; ok {
			amount = new(big.Int).Add(userAmount, feeAmount)
			feeOccupy = new(big.Float).Quo(new(big.Float).SetInt(feeAmount), new(big.Float).SetInt(amount)).String()
		} else {
			amount = userAmount
		}
		if tokenId == VxTokenId {
			foundVx = true
			amount = amount.Add(amount, vxAmount)
		}
		balance, _ = db.GetBalance(&tokenId)
		if balanceMatched = amount.Cmp(balance) == 0; !balanceMatched {
			allBalanceMatched = false
		}
		verifyItems[tokenId] = &FundVerifyItem{TokenId: tokenId, Balance: balance.String(), Amount: amount.String(), UserAmount: userAmount.String(), FeeAmount: feeAmount.String(), FeeOccupy: feeOccupy, BalanceMatched: balanceMatched}
	}
	if !foundVx && vxAmount.Sign() > 0 {
		balance, _ = db.GetBalance(&VxTokenId)
		if balanceMatched = vxAmount.Cmp(balance) == 0; !balanceMatched {
			allBalanceMatched = false
		}
		verifyItems[VxTokenId] = &FundVerifyItem{TokenId: VxTokenId, Balance: balance.String(), Amount: vxAmount.String(), UserAmount: "0", FeeAmount: "0", FeeOccupy: "0", BalanceMatched: balanceMatched}
	}
	return &FundVerifyRes{
		count,
		allBalanceMatched,
		verifyItems,
	}
}

func accumulateUserAccount(db vm_db.VmDb, accumulateRes map[types.TokenTypeId]*big.Int) (int, error) {
	var (
		userAccountValue []byte
		userFund         *UserFund
		ok               bool
	)
	var count = 0
	iterator, err := db.NewStorageIterator(fundKeyPrefix)
	if err != nil {
		return 0, err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			userAccountValue = iterator.Value()
			if len(userAccountValue) == 0 {
				continue
			}
		} else {
			break
		}
		userFund = &UserFund{}
		if err = userFund.DeSerialize(userAccountValue); err != nil {
			return 0, err
		}
		for _, acc := range userFund.Accounts {
			tokenId, _ := types.BytesToTokenTypeId(acc.Token)
			total := AddBigInt(acc.Available, acc.Locked)
			accAccount(tokenId, total, accumulateRes)
		}
		count++
	}
	return count, nil
}

func accumulateFeeDividendPool(db vm_db.VmDb, reader *util.VMConsensusReader, accumulateRes map[types.TokenTypeId]*big.Int) error {
	var (
		feeSumValue, feeSumKey []byte
		feeSum                 *FeeSumByPeriod
		ok                     bool
		periodId               uint64
	)
	currentPeriodId := GetCurrentPeriodId(db, reader)
	iterator, err := db.NewStorageIterator(feeSumKeyPrefix)
	if err != nil {
		return err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			feeSumKey = iterator.Key()
			feeSumValue = iterator.Value()
			if len(feeSumValue) == 0 {
				continue
			}
			periodId = BytesToUint64(feeSumKey[len(feeSumKeyPrefix):])
		} else {
			break
		}
		feeSum = &FeeSumByPeriod{}
		if err = feeSum.DeSerialize(feeSumValue); err != nil {
			return err
		}
		if !feeSum.FinishFeeDividend {
			for _, fee := range feeSum.FeesForDividend {
				tokenId, _ := types.BytesToTokenTypeId(fee.Token)
				if currentPeriodId != periodId {
					toDividend, _ := splitDividendPool(fee)
					accAccount(tokenId, toDividend.Bytes(), accumulateRes)
				} else {
					accAccount(tokenId, fee.DividendPoolAmount, accumulateRes)
				}
			}
		}
	}
	return nil
}

func accumulateBrokerFeeAccount(db vm_db.VmDb, accumulateRes map[types.TokenTypeId]*big.Int) error {
	var (
		brokerFeeSumValue []byte
		brokerFeeSum      *BrokerFeeSumByPeriod
		ok                bool
	)
	iterator, err := db.NewStorageIterator(brokerFeeSumKeyPrefix)
	if err != nil {
		return err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			brokerFeeSumValue = iterator.Value()
			if len(brokerFeeSumValue) == 0 {
				continue
			}
		} else {
			break
		}
		brokerFeeSum = &BrokerFeeSumByPeriod{}
		if err = brokerFeeSum.DeSerialize(brokerFeeSumValue); err != nil {
			return err
		}
		for _, fee := range brokerFeeSum.BrokerFees {
			for _, brokerFee := range fee.MarketFees {
				tokenId, _ := types.BytesToTokenTypeId(fee.Token)
				accAccount(tokenId, brokerFee.Amount, accumulateRes)
			}
		}
	}
	return nil
}

func accumulateVx(db vm_db.VmDb, vxAmount *big.Int) error {
	var (
		amtBytes []byte
		ok       bool
	)
	vxAmount.Add(vxAmount, GetVxMinePool(db))
	iterator, err := db.NewStorageIterator(makerMineProxyAmountByPeriodKey)
	if err != nil {
		return err
	}
	defer iterator.Release()
	for {
		if ok = iterator.Next(); ok {
			amtBytes = iterator.Value()
			if len(amtBytes) == 0 {
				continue
			}
		} else {
			break
		}
		amt := new(big.Int).SetBytes(amtBytes)
		vxAmount.Add(vxAmount, amt)
	}
	return nil
}

func accAccount(tokenId types.TokenTypeId, amount []byte, accAccount map[types.TokenTypeId]*big.Int) {
	if acc, ok := accAccount[tokenId]; ok {
		acc.Add(acc, new(big.Int).SetBytes(amount))
	} else {
		accAccount[tokenId] = new(big.Int).SetBytes(amount)
	}
}
