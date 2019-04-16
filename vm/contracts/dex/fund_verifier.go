package dex

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

type FundVerifyRes struct {
	TokenId    types.TokenTypeId `json:"tokenId"`
	Balance    string            `json:"balance"`
	Amount     string            `json:"amount"`
	UserAmount string            `json:"userAmount"`
	FeeAmount  string            `json:"feeAmount"`
	FeeOccupy  string            `json:"feeOccupy"`
	Ok         bool              `json:"ok"`
}

func VerifyDexFundBalance(db vmctxt_interface.VmDatabase) map[types.TokenTypeId]*FundVerifyRes {
	userAmountMap := make(map[types.TokenTypeId]*big.Int)
	feeAmountMap := make(map[types.TokenTypeId]*big.Int)
	showRes := make(map[types.TokenTypeId]*FundVerifyRes)
	accumulateUserAccount(db, userAmountMap)
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
		showRes[tokenId] = &FundVerifyRes{TokenId: tokenId, Balance: balance.String(), Amount: amount.String(), UserAmount: userAmount.String(), FeeAmount: feeAmount.String(), FeeOccupy: feeOccupy, Ok: amount.Cmp(balance) == 0}
	}
	return showRes
}

func accumulateUserAccount(db vmctxt_interface.VmDatabase, accumulateRes map[types.TokenTypeId]*big.Int) (error) {
	var (
		userAccountValue []byte
		userFund         *UserFund
		err              error
		ok               bool
	)
	iterator := db.NewStorageIterator(&types.AddressDexFund, fundKeyPrefix)
	for {
		if _, userAccountValue, ok = iterator.Next(); !ok {
			break
		}
		userFund = &UserFund{}
		if userFund, err = userFund.DeSerialize(userAccountValue); err != nil {
			return err
		}
		for _, acc := range userFund.Accounts {
			tokenId, _ := types.BytesToTokenTypeId(acc.Token)
			total := AddBigInt(acc.Available, acc.Locked)
			accAccount(tokenId, total, accumulateRes)
		}
	}
	return nil
}

func accumulateFeeAccount(db vmctxt_interface.VmDatabase, accumulateRes map[types.TokenTypeId]*big.Int) error {
	var (
		feeSumValue, donateFeeSumValue []byte
		feeSum      *FeeSumByPeriod
		err         error
		ok          bool
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
