package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type FundVerifyRes struct {
	UserCount   int                                   `json:"userCount"`
	Ok          bool                                  `json:"ok"`
	VerifyItems map[types.TokenTypeId]*FundVerifyItem `json:"balances"`
}

type FundVerifyItem struct {
	TokenId    types.TokenTypeId `json:"tokenId"`
	Balance    string            `json:"balance"`
	Amount     string            `json:"amount"`
	UserAmount string            `json:"userAmount"`
	UserCount  int               `json:"userCount"`
	FeeAmount  string            `json:"feeAmount"`
	FeeOccupy  string            `json:"feeOccupy"`
	Ok         bool              `json:"ok"`
}

func VerifyDexFundBalance(db vm_db.VmDb) *FundVerifyRes {
	userAmountMap := make(map[types.TokenTypeId]*big.Int)
	feeAmountMap := make(map[types.TokenTypeId]*big.Int)
	verifyItems := make(map[types.TokenTypeId]*FundVerifyItem)
	count, _ := accumulateUserAccount(db, userAmountMap)
	balanceMatch := true
	accumulateFeeAccount(db, feeAmountMap)
	for tokenId, userAmount := range userAmountMap {
		var (
			amount    *big.Int
			feeAmount *big.Int
			ok        bool
			feeOccupy string
		)
		if feeAmount, ok = feeAmountMap[tokenId]; ok {
			amount = new(big.Int).Add(userAmount, feeAmount)
			feeOccupy = new(big.Float).Quo(new(big.Float).SetInt(feeAmount), new(big.Float).SetInt(amount)).String()
		} else {
			amount = userAmount
		}
		balance := db.GetBalance(&types.AddressDexFund, &tokenId)
		ok = amount.Cmp(balance) == 0
		verifyItems[tokenId] = &FundVerifyItem{TokenId: tokenId, Balance: balance.String(), Amount: amount.String(), UserAmount: userAmount.String(), FeeAmount: feeAmount.String(), FeeOccupy: feeOccupy, Ok: ok}
		if !ok {
			balanceMatch = false
		}
	}
	return &FundVerifyRes{
		count,
		balanceMatch,
		verifyItems,
	}
}

func accumulateUserAccount(db vm_db.VmDb, accumulateRes map[types.TokenTypeId]*big.Int) (int, error) {
	var (
		userAccountValue []byte
		userFund         *UserFund
		err              error
		ok               bool
	)
	var count = 0
	iterator := db.NewStorageIterator(&types.AddressDexFund, fundKeyPrefix)
	for {
		if _, userAccountValue, ok = iterator.Next(); !ok {
			break
		}
		userFund = &UserFund{}
		if userFund, err = userFund.DeSerialize(userAccountValue); err != nil {
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

func accumulateFeeAccount(db vm_db.VmDb, accumulateRes map[types.TokenTypeId]*big.Int) error {
	var (
		feeSumValue, donateFeeSumValue []byte
		feeSum                         *FeeSumByPeriod
		err                            error
		ok                             bool
	)
	iterator := db.NewStorageIterator(&types.AddressDexFund, feeSumKeyPrefix)
	for {
		if _, feeSumValue, ok = iterator.Next(); !ok {
			break
		}
		feeSum = &FeeSumByPeriod{}
		if feeSum, err = feeSum.DeSerialize(feeSumValue); err != nil {
			return err
		}
		if !feeSum.FeeDivided {
			for _, fee := range feeSum.Fees {
				tokenId, _ := types.BytesToTokenTypeId(fee.Token)
				accAccount(tokenId, fee.Amount, accumulateRes)
			}
		}
	}
	iterator = db.NewStorageIterator(&types.AddressDexFund, donateFeeSumKeyPrefix)
	for {
		if _, donateFeeSumValue, ok = iterator.Next(); !ok {
			break
		}
		accAccount(ledger.ViteTokenId, donateFeeSumValue, accumulateRes)
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
