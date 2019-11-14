package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func DoSettleFund(db vm_db.VmDb, reader util.ConsensusReader, action *dexproto.FundSettle, marketInfo *MarketInfo, fundLogger log15.Logger) error {
	address := types.Address{}
	address.SetBytes([]byte(action.Address))
	dexFund, _ := GetFund(db, address)
	for _, accountSettle := range action.AccountSettles {
		var token []byte
		if accountSettle.IsTradeToken {
			token = marketInfo.TradeToken
		} else {
			token = marketInfo.QuoteToken
		}
		if tokenId, err := types.BytesToTokenTypeId(token); err != nil {
			return err
		} else {
			if _, ok := GetTokenInfo(db, tokenId); !ok {
				panic(InvalidTokenErr)
			}
			account, exists := GetAccountByToken(dexFund, tokenId)
			var (
				exceed    bool
				actualSub []byte
			)
			//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
			if CmpToBigZero(accountSettle.ReduceLocked) != 0 {
				if account.Locked, _, exceed = SafeSubBigInt(account.Locked, accountSettle.ReduceLocked); exceed {
					if IsDexFeeFork(db) {
						fundLogger.Error(cabi.MethodNameDexFundSettleOrdersV2+" DoSettleFund exceed for reduceLocked", "locked", new(big.Int).SetBytes(account.Locked).String(), "reduceLocked", new(big.Int).SetBytes(accountSettle.ReduceLocked).String())
					} else {
						panic(ExceedFundLockedErr)
					}
				}
			}
			if CmpToBigZero(accountSettle.ReleaseLocked) != 0 {
				if account.Locked, actualSub, exceed = SafeSubBigInt(account.Locked, accountSettle.ReleaseLocked); exceed {
					if IsDexFeeFork(db) {
						fundLogger.Error(cabi.MethodNameDexFundSettleOrdersV2+" DoSettleFund exceed for releaseLocked", "locked", new(big.Int).SetBytes(account.Locked).String(), "releaseLocked", new(big.Int).SetBytes(accountSettle.ReleaseLocked).String())
					} else {
						panic(ExceedFundLockedErr)
					}
				}
				account.Available = AddBigInt(account.Available, actualSub)
			}
			if CmpToBigZero(accountSettle.IncAvailable) != 0 {
				account.Available = AddBigInt(account.Available, accountSettle.IncAvailable)
			}
			if !exists {
				dexFund.Accounts = append(dexFund.Accounts, account)
			}
			// must do after account updated by settle
			if bytes.Equal(token, VxTokenId.Bytes()) {
				if err = OnSettleVx(db, reader, action.Address, accountSettle, account); err != nil {
					return err
				}
			}
			//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
		}
	}
	SaveFund(db, address, dexFund)
	return nil
}

func SettleFees(db vm_db.VmDb, reader util.ConsensusReader, allowMining bool, feeToken []byte, feeTokenDecimals, quoteTokenType int32, feeActions []*dexproto.FeeSettle, feeForDividend *big.Int, inviteRelations map[types.Address]*types.Address) {
	tokenId, _ := types.BytesToTokenTypeId(feeToken)
	SettleFeesWithTokenId(db, reader, allowMining, tokenId, feeTokenDecimals, quoteTokenType, feeActions, feeForDividend, inviteRelations)
}

func SettleFeesWithTokenId(db vm_db.VmDb, reader util.ConsensusReader, allowMining bool, tokenId types.TokenTypeId, feeTokenDecimals, quoteTokenType int32, feeActions []*dexproto.FeeSettle, feeForDividend *big.Int, inviteRelations map[types.Address]*types.Address) {
	if len(feeActions) == 0 && feeForDividend == nil {
		return
	}
	currentPeriodId := GetCurrentPeriodId(db, reader)
	dexFees, ok := GetDexFeesByPeriodId(db, currentPeriodId)
	if !ok { // need roll period when current period dexFees not saved yet
		dexFees = RollAndGentNewDexFeesByPeriod(db, currentPeriodId)
	}
	if inviteRelations == nil {
		inviteRelations = make(map[types.Address]*types.Address)
	}
	var incBaseSumForDividend []byte
	var needIncSumForMine bool
	var incBaseSumForMine, incInviteeSumForMine []byte
	var mineThreshold *big.Int
	if allowMining {
		mineThreshold = GetMineThreshold(db, quoteTokenType)
	}

	for _, feeAction := range feeActions {
		incBaseSumForDividend = AddBigInt(incBaseSumForDividend, feeAction.BaseFee)
		if allowMining {
			var needAddSum bool
			var addBaseSum, addInviteeSum []byte
			inviteRelations, needAddSum, addBaseSum, addInviteeSum = settleUserFees(db, currentPeriodId, feeTokenDecimals, quoteTokenType, mineThreshold, feeAction, inviteRelations)
			if needAddSum {
				needIncSumForMine = true
				incBaseSumForMine = AddBigInt(incBaseSumForMine, addBaseSum)
				incInviteeSumForMine = AddBigInt(incInviteeSumForMine, addInviteeSum)
			}
		}
	}
	//fmt.Printf("needIncSumForMine %v, incBaseSumForMine %s, incInviteeSumForMine %s\n", needIncSumForMine, new(big.Int).SetBytes(incBaseSumForMine).String(),
	//	new(big.Int).SetBytes(incInviteeSumForMine).String())

	// settle dividend fee
	var foundDividendFeeToken bool
	for _, dividendAcc := range dexFees.FeesForDividend {
		if bytes.Equal(tokenId.Bytes(), dividendAcc.Token) {
			dividendAcc.DividendPoolAmount = AddBigInt(dividendAcc.DividendPoolAmount, incBaseSumForDividend)
			if feeForDividend != nil {
				dividendAcc.DividendPoolAmount = AddBigInt(dividendAcc.DividendPoolAmount, feeForDividend.Bytes())
			}
			foundDividendFeeToken = true
			break
		}
	}
	if !foundDividendFeeToken {
		dexFees.FeesForDividend = append(dexFees.FeesForDividend, newFeesForDividend(tokenId.Bytes(), incBaseSumForDividend, feeForDividend))
	}
	// settle mine fee
	if needIncSumForMine {
		var foundMineFeeTokenType bool
		for _, mineAcc := range dexFees.FeesForMine {
			if quoteTokenType == mineAcc.QuoteTokenType {
				mineAcc.BaseAmount = AddBigInt(mineAcc.BaseAmount, incBaseSumForMine)
				if len(incInviteeSumForMine) > 0 {
					mineAcc.InviteBonusAmount = AddBigInt(mineAcc.InviteBonusAmount, incInviteeSumForMine)
				}
				foundMineFeeTokenType = true
				break
			}
		}
		if !foundMineFeeTokenType {
			dexFees.FeesForMine = append(dexFees.FeesForMine, newFeesForMine(quoteTokenType, incBaseSumForMine, incInviteeSumForMine))
		}
	}
	SaveDexFeesByPeriodId(db, currentPeriodId, dexFees)
}

//baseAmount + operatorAmount for vx mine,
func settleUserFees(db vm_db.VmDb, periodId uint64, tokenDecimals, quoteTokenType int32, mineThreshold *big.Int, feeAction *dexproto.FeeSettle, inviteRelations map[types.Address]*types.Address) (map[types.Address]*types.Address, bool, []byte, []byte) {
	if inviteRelations == nil {
		inviteRelations = make(map[types.Address]*types.Address)
	}
	isInvited, inviter, inviterBonus, inviteeBonus := getInviteBonusInfo(db, feeAction.Address, &inviteRelations, feeAction.BaseFee)
	if isInvited && !IsEarthFork(db) {
		inviteeBonus = nil
	}
	needAddSum, addBaseSum, addInviteeSum := innerSettleUserFee(db, periodId, mineThreshold, feeAction.Address, tokenDecimals, quoteTokenType, feeAction.BaseFee, inviteeBonus)
	if isInvited {
		if neeAddSum1, addBaseSum1, addInviteeSum1 := innerSettleUserFee(db, periodId, mineThreshold, inviter.Bytes(), tokenDecimals, quoteTokenType, nil, inviterBonus); neeAddSum1 {
			needAddSum = true
			addBaseSum = AddBigInt(addBaseSum, addBaseSum1)
			addInviteeSum = AddBigInt(addInviteeSum, addInviteeSum1)
		}
	}
	return inviteRelations, needAddSum, addBaseSum, addInviteeSum
}

func innerSettleUserFee(db vm_db.VmDb, periodId uint64, mineThreshold *big.Int, address []byte, tokenDecimals, quoteTokenType int32, baseTokenFee, inviteBonusTokenFee []byte) (needAddSum bool, addBaseSumNormalAmt, addInviteeSumNormalAmt []byte) {
	userFees, _ := GetUserFees(db, address)
	feeLen := len(userFees.Fees)
	addBaseSumNormalAmt = NormalizeToQuoteTokenTypeAmount(baseTokenFee, tokenDecimals, quoteTokenType)
	addInviteeSumNormalAmt = NormalizeToQuoteTokenTypeAmount(inviteBonusTokenFee, tokenDecimals, quoteTokenType)
	if feeLen > 0 && periodId == userFees.Fees[feeLen-1].Period {
		var foundToken = false
		for _, userFee := range userFees.Fees[feeLen-1].Fees {
			if userFee.QuoteTokenType == quoteTokenType {
				originValid := IsValidFeeForMine(userFee, mineThreshold)
				if addBaseSumNormalAmt != nil {
					userFee.BaseAmount = AddBigInt(userFee.BaseAmount, addBaseSumNormalAmt)
				}
				if addInviteeSumNormalAmt != nil {
					userFee.InviteBonusAmount = AddBigInt(userFee.InviteBonusAmount, addInviteeSumNormalAmt)
				}
				needAddSum = IsValidFeeForMine(userFee, mineThreshold)
				if needAddSum && !originValid {
					addBaseSumNormalAmt = userFee.BaseAmount
					addInviteeSumNormalAmt = userFee.InviteBonusAmount
				}
				foundToken = true
				break
			}
		}
		if !foundToken {
			uf := newFeeAccount(quoteTokenType, addBaseSumNormalAmt, addInviteeSumNormalAmt)
			userFees.Fees[feeLen-1].Fees = append(userFees.Fees[feeLen-1].Fees, uf)
			needAddSum = IsValidFeeForMine(uf, mineThreshold)
		}
	} else {
		userFeeByPeriodId := &dexproto.FeesByPeriod{}
		userFeeByPeriodId.Period = periodId
		uf1 := newFeeAccount(quoteTokenType, addBaseSumNormalAmt, addInviteeSumNormalAmt)
		userFeeByPeriodId.Fees = []*dexproto.FeeAccount{uf1}
		userFees.Fees = append(userFees.Fees, userFeeByPeriodId)
		needAddSum = IsValidFeeForMine(uf1, mineThreshold)
	}
	if !needAddSum {
		addBaseSumNormalAmt = nil
		addInviteeSumNormalAmt = nil
	}
	SaveUserFees(db, address, userFees)
	//addr, _ := types.BytesToAddress(address)
	//fmt.Printf("addr %s, needAddSum %v, addBaseSumNormalAmt %s, addInviteeSumNormalAmt %s\n", addr.String(), needAddSum, new(big.Int).SetBytes(addBaseSumNormalAmt).String(),
	//	new(big.Int).SetBytes(addInviteeSumNormalAmt).String())

	return
}

func SettleOperatorFees(db vm_db.VmDb, reader util.ConsensusReader, feeActions []*dexproto.FeeSettle, marketInfo *MarketInfo) {
	var (
		incAmt               []byte
		operatorFeesByPeriod *OperatorFeesByPeriod
		hasValidAmount       bool
	)

	for _, feeAction := range feeActions {
		if len(feeAction.OperatorFee) > 0 && CmpToBigZero(feeAction.OperatorFee) > 0 {
			incAmt = AddBigInt(incAmt, feeAction.OperatorFee)
			hasValidAmount = true
		}
	}
	if !hasValidAmount {
		return
	}
	operatorFeesByPeriod, _ = GetCurrentOperatorFees(db, reader, marketInfo.Owner)
	var foundToken bool
	for _, operatorFee := range operatorFeesByPeriod.OperatorFees {
		if bytes.Equal(marketInfo.QuoteToken, operatorFee.Token) {
			var foundMarket bool
			for _, mkFee := range operatorFee.MarketFees {
				if mkFee.MarketId == marketInfo.MarketId {
					incOperatorMarketFee(marketInfo, mkFee, incAmt)
					foundMarket = true
					break
				}
			}
			if !foundMarket {
				operatorFee.MarketFees = append(operatorFee.MarketFees, newOperatorMarketFee(marketInfo, incAmt))
			}
			foundToken = true
			break
		}
	}
	if !foundToken {
		feeAcc := &dexproto.OperatorFeeAccount{}
		feeAcc.Token = marketInfo.QuoteToken
		feeAcc.MarketFees = append(feeAcc.MarketFees, newOperatorMarketFee(marketInfo, incAmt))
		operatorFeesByPeriod.OperatorFees = append(operatorFeesByPeriod.OperatorFees, feeAcc)
	}
	SaveCurrentOperatorFees(db, reader, marketInfo.Owner, operatorFeesByPeriod)
}

func OnDepositVx(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, depositAmount *big.Int, updatedVxAccount *dexproto.Account) error {
	if IsEarthFork(db) {
		return nil
	} else {
		return DoSettleVxFunds(db, reader, address.Bytes(), depositAmount, updatedVxAccount)
	}
}

func OnWithdrawVx(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, withdrawAmount *big.Int, updatedVxAccount *dexproto.Account) error {
	if IsEarthFork(db) {
		return nil
	} else {
		return DoSettleVxFunds(db, reader, address.Bytes(), new(big.Int).Neg(withdrawAmount), updatedVxAccount)
	}
}

func OnSettleVx(db vm_db.VmDb, reader util.ConsensusReader, address []byte, fundSettle *dexproto.AccountSettle, updatedVxAccount *dexproto.Account) error {
	if IsEarthFork(db) {
		return nil
	} else {
		amtChange := SubBigInt(fundSettle.IncAvailable, fundSettle.ReduceLocked)
		return DoSettleVxFunds(db, reader, address, amtChange, updatedVxAccount)
	}
}

func OnVxMined(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, amount *big.Int) error {
	if IsEarthFork(db) {
		if IsAutoLockMinedVx(db, address.Bytes()) {
			updatedVxAccount := LockMinedVx(db, address, amount)
			return DoSettleVxFunds(db, reader, address.Bytes(), amount, updatedVxAccount)
		} else {
			DepositAccount(db, address, VxTokenId, amount)
		}
	} else {
		updatedVxAccount := DepositAccount(db, address, VxTokenId, amount)
		DoSettleVxFunds(db, reader, address.Bytes(), amount, updatedVxAccount)
	}
	return nil
}

// only settle validAmount and amount changed from previous period
func DoSettleVxFunds(db vm_db.VmDb, reader util.ConsensusReader, addressBytes []byte, amtChange *big.Int, updatedVxAccount *dexproto.Account) error {
	var (
		vxFunds               *VxFunds
		userNewAmt, sumChange *big.Int
		periodId              uint64
		originFundsLen        int
		needUpdate            bool
	)
	vxFunds, _ = GetVxFundsWithForkCheck(db, addressBytes)
	periodId = GetCurrentPeriodId(db, reader)
	originFundsLen = len(vxFunds.Funds)
	userNewAmt = getUserNewVxAmtWithForkCheck(db, updatedVxAccount)
	if originFundsLen == 0 { //need append new period
		if IsValidVxAmountForDividend(userNewAmt) {
			fundByPeriod := &dexproto.VxFundByPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			vxFunds.Funds = append(vxFunds.Funds, fundByPeriod)
			sumChange = userNewAmt
			needUpdate = true
		}
	} else if vxFunds.Funds[originFundsLen-1].Period == periodId { //update current period
		if IsValidVxAmountForDividend(userNewAmt) {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[originFundsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = userNewAmt
			}
			vxFunds.Funds[originFundsLen-1].Amount = userNewAmt.Bytes()
		} else {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[originFundsLen-1].Amount) {
				sumChange = NegativeAmount(vxFunds.Funds[originFundsLen-1].Amount)
			}
			if originFundsLen > 1 { // in case originFundsLen > 1, update last period to diff the condition of current period not changed ever from last saved period
				vxFunds.Funds[originFundsLen-1].Amount = userNewAmt.Bytes()
			} else { // clear funds in case only current period saved and not valid any more
				vxFunds.Funds = nil
			}
		}
		needUpdate = true
	} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
		if IsValidVxAmountForDividend(userNewAmt) {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[originFundsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = userNewAmt
			}
			fundWithPeriod := &dexproto.VxFundByPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
			needUpdate = true
		} else {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[originFundsLen-1].Amount) {
				sumChange = NegativeAmount(vxFunds.Funds[originFundsLen-1].Amount)
				fundWithPeriod := &dexproto.VxFundByPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
				vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
				needUpdate = true
			}
		}
	}

	if len(vxFunds.Funds) > 0 && needUpdate {
		SaveVxFundsWithForkCheck(db, addressBytes, vxFunds)
	} else if len(vxFunds.Funds) == 0 && originFundsLen > 0 {
		DeleteVxFundsWithForkCheck(db, addressBytes)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		vxSumFunds, _ := GetVxSumFundsWithForkCheck(db)
		sumFundsLen := len(vxSumFunds.Funds)
		if sumFundsLen == 0 {
			if sumChange.Sign() > 0 {
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundByPeriod{Period: periodId, Amount: sumChange.Bytes()})
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
				// roll new period
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundByPeriod{Amount: sumRes.Bytes(), Period: periodId})
			}
		}
		SaveVxSumFundsWithForkCheck(db, vxSumFunds)
	}
	return nil
}

func getUserNewVxAmtWithForkCheck(db vm_db.VmDb, updatedVxAcc *dexproto.Account) *big.Int {
	if IsEarthFork(db) {
		return new(big.Int).SetBytes(updatedVxAcc.VxLocked)
	} else {
		return new(big.Int).SetBytes(AddBigInt(updatedVxAcc.Available, updatedVxAcc.Locked))
	}
}

func splitDividendPool(dividend *dexproto.FeeForDividend) (toDividendAmt, rolledAmount *big.Int) {
	if !dividend.NotRoll {
		toDividendAmt = new(big.Int).SetBytes(CalculateAmountForRate(dividend.DividendPoolAmount, PerPeriodDividendRate)) // %1
		rolledAmount = new(big.Int).Sub(new(big.Int).SetBytes(dividend.DividendPoolAmount), toDividendAmt)                // 99%
	} else {
		toDividendAmt = new(big.Int).SetBytes(dividend.DividendPoolAmount)
		rolledAmount = big.NewInt(0)
	}
	return
}

func getInviteBonusInfo(db vm_db.VmDb, addr []byte, inviteRelations *map[types.Address]*types.Address, fee []byte) (bool, *types.Address, []byte, []byte) {
	if address, err := types.BytesToAddress(addr); err != nil {
		panic(InternalErr)
	} else {
		var (
			inviter *types.Address
			ok      bool
		)
		if inviter, ok = (*inviteRelations)[address]; !ok {
			if inviter, err = GetInviterByInvitee(db, address); err == nil {
				(*inviteRelations)[address] = inviter
			} else if err == NotBindInviterErr {
				(*inviteRelations)[address] = nil
			} else {
				panic(InternalErr)
			}
		}
		if inviter != nil {
			return true, inviter, CalculateAmountForRate(fee, InviterBonusRate), CalculateAmountForRate(fee, InviteeBonusRate)
		} else {
			return false, nil, nil, nil
		}
	}
}

func newFeeAccount(quoteTokenType int32, baseAmount, inviteBonusAmount []byte) *dexproto.FeeAccount {
	account := &dexproto.FeeAccount{}
	account.QuoteTokenType = quoteTokenType
	account.BaseAmount = baseAmount
	account.InviteBonusAmount = inviteBonusAmount
	return account
}

func newFeesForDividend(token, initAmount []byte, dividendAmt *big.Int) *dexproto.FeeForDividend {
	account := &dexproto.FeeForDividend{}
	account.Token = token
	account.DividendPoolAmount = initAmount
	if dividendAmt != nil {
		account.DividendPoolAmount = AddBigInt(account.DividendPoolAmount, dividendAmt.Bytes())
	}
	return account
}

func newFeesForMine(quoteTokenType int32, baseAmount, inviteBonusAmount []byte) *dexproto.FeeForMine {
	account := &dexproto.FeeForMine{}
	account.QuoteTokenType = quoteTokenType
	account.BaseAmount = baseAmount
	account.InviteBonusAmount = inviteBonusAmount
	return account
}

func newOperatorMarketFee(marketInfo *MarketInfo, amount []byte) *dexproto.OperatorMarketFee {
	account := &dexproto.OperatorMarketFee{}
	account.MarketId = marketInfo.MarketId
	account.TakerOperatorFeeRate = marketInfo.TakerOperatorFeeRate
	account.MakerOperatorFeeRate = marketInfo.MakerOperatorFeeRate
	account.Amount = amount
	return account
}

func incOperatorMarketFee(marketInfo *MarketInfo, marketFee *dexproto.OperatorMarketFee, incAmt []byte) {
	marketFee.TakerOperatorFeeRate = marketInfo.TakerOperatorFeeRate
	marketFee.MakerOperatorFeeRate = marketInfo.MakerOperatorFeeRate
	marketFee.Amount = AddBigInt(marketFee.Amount, incAmt)
}
