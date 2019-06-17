package dex

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func DoSettleFund(db vm_db.VmDb, reader util.ConsensusReader, action *dexproto.UserFundSettle, marketInfo *MarketInfo) error {
	address := types.Address{}
	address.SetBytes([]byte(action.Address))
	dexFund, _ := GetUserFund(db, address)
	for _, fundSettle := range action.FundSettles {
		var token []byte
		if fundSettle.IsTradeToken {
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
			account, exists := GetAccountByTokeIdFromFund(dexFund, tokenId)
			//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
			if CmpToBigZero(fundSettle.ReduceLocked) != 0 {
				if CmpForBigInt(fundSettle.ReduceLocked, account.Locked) > 0 {
					panic(ExceedFundLockedErr)
				}
				account.Locked = SubBigIntAbs(account.Locked, fundSettle.ReduceLocked)
			}
			if CmpToBigZero(fundSettle.ReleaseLocked) != 0 {
				if CmpForBigInt(fundSettle.ReleaseLocked, account.Locked) > 0 {
					panic(ExceedFundLockedErr)
				}
				account.Locked = SubBigIntAbs(account.Locked, fundSettle.ReleaseLocked)
				account.Available = AddBigInt(account.Available, fundSettle.ReleaseLocked)
			}
			if CmpToBigZero(fundSettle.IncAvailable) != 0 {
				account.Available = AddBigInt(account.Available, fundSettle.IncAvailable)
			}
			if !exists {
				dexFund.Accounts = append(dexFund.Accounts, account)
			}
			// must do after account updated by settle
			if bytes.Equal(token, VxTokenBytes) {
				OnSettleVx(db, reader, action.Address, fundSettle, account)
			}
			//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
		}
	}
	SaveUserFund(db, address, dexFund)
	return nil
}

func SettleFeeSum(db vm_db.VmDb, reader util.ConsensusReader, allowMine bool, feeToken []byte, feeTokenDecimals, quoteTokenType int32, feeActions []*dexproto.UserFeeSettle, feeForDividend *big.Int, inviteRelations map[types.Address]*types.Address) map[types.Address]*types.Address {
	tokenId, _ := types.BytesToTokenTypeId(feeToken)
	return SettleFeeSumWithTokenId(db, reader, allowMine, tokenId, feeTokenDecimals, quoteTokenType, feeActions, feeForDividend, inviteRelations)
}

func SettleFeeSumWithTokenId(db vm_db.VmDb, reader util.ConsensusReader, allowMine bool, tokenId types.TokenTypeId, feeTokenDecimals, quoteTokenType int32, feeActions []*dexproto.UserFeeSettle, feeForDividend *big.Int, inviteRelations map[types.Address]*types.Address) map[types.Address]*types.Address {
	feeSumByPeriod, ok := GetCurrentFeeSum(db, reader)
	if !ok { // need roll period when current period feeSum not saved yet
		feeSumByPeriod = rollFeeSum(db, reader)
	}
	if len(feeActions) > 0 {
		if inviteRelations == nil {
			inviteRelations = make(map[types.Address]*types.Address)
		}
		var incBaseFeeAmt, incInviteBonusAmt []byte
		for _, feeAction := range feeActions {
			incBaseFeeAmt = AddBigInt(incBaseFeeAmt, feeAction.BaseFee)
			if allowMine {
				if isInvited, _, inviteBonusAmt := getInviteBonusInfo(db, feeAction.Address, &inviteRelations, feeAction.BaseFee); isInvited {
					incInviteBonusAmt = AddBigInt(incInviteBonusAmt, inviteBonusAmt)
				}
			}
		}
		// settle dividend fee
		var foundDividendFeeToken bool
		for _, dividendAcc := range feeSumByPeriod.FeesForDividend {
			if bytes.Equal(tokenId.Bytes(), dividendAcc.Token) {
				if allowMine {
					dividendAcc.BaseAmount = AddBigInt(dividendAcc.BaseAmount, incBaseFeeAmt)
				} else {
					dividendAcc.DividendPoolAmount = AddBigInt(dividendAcc.DividendPoolAmount, incBaseFeeAmt)
				}
				foundDividendFeeToken = true
				break
			}
		}
		if !foundDividendFeeToken {
			if allowMine {
				feeSumByPeriod.FeesForDividend = append(feeSumByPeriod.FeesForDividend, newFeeSumForDividend(tokenId.Bytes(), incBaseFeeAmt, nil))
			} else {
				feeSumByPeriod.FeesForDividend = append(feeSumByPeriod.FeesForDividend, newFeeSumForDividend(tokenId.Bytes(), nil, incBaseFeeAmt))
			}
		}
		// settle mine fee
		if allowMine {
			var foundMineFeeTokenType bool
			for _, mineAcc := range feeSumByPeriod.FeesForMine {
				if quoteTokenType == mineAcc.QuoteTokenType {
					mineAcc.BaseAmount = AddBigInt(mineAcc.BaseAmount, AdjustAmountToQuoteTokenType(incBaseFeeAmt, feeTokenDecimals, quoteTokenType).Bytes())
					if len(incInviteBonusAmt) > 0 {
						mineAcc.InviteBonusAmount = AddBigInt(mineAcc.InviteBonusAmount, incInviteBonusAmt)
					}
					foundMineFeeTokenType = true
					break
				}
			}
			if !foundMineFeeTokenType {
				feeSumByPeriod.FeesForMine = append(feeSumByPeriod.FeesForMine, newFeeSumForMine(quoteTokenType, incBaseFeeAmt, incInviteBonusAmt))
			}
		}
	}
	if feeForDividend != nil {
		var foundToken bool
		for _, feeAcc := range feeSumByPeriod.FeesForDividend {
			if bytes.Equal(feeAcc.Token, tokenId.Bytes()) {
				feeAcc.DividendPoolAmount = AddBigInt(feeAcc.DividendPoolAmount, feeForDividend.Bytes())
				foundToken = true
			}
		}
		if !foundToken {
			feeSumByPeriod.FeesForDividend = append(feeSumByPeriod.FeesForDividend, newFeeSumForDividend(tokenId.Bytes(), nil, feeForDividend.Bytes()))
		}
	}
	SaveCurrentFeeSum(db, reader, feeSumByPeriod)
	return inviteRelations
}

//baseAmount + brokerAmount for vx mine,
func SettleUserFees(db vm_db.VmDb, reader util.ConsensusReader, tokenDecimals, quoteTokenType int32, feeAction *dexproto.UserFeeSettle, inviteRelations map[types.Address]*types.Address) map[types.Address]*types.Address {
	if inviteRelations == nil {
		inviteRelations = make(map[types.Address]*types.Address)
	}
	innerSettleUserFee(db, reader, feeAction.Address, tokenDecimals, quoteTokenType, feeAction.BaseFee, nil)
	if isInvited, inviter, inviteBonusAmt := getInviteBonusInfo(db, feeAction.Address, &inviteRelations, feeAction.BaseFee); isInvited {
		innerSettleUserFee(db, reader, inviter.Bytes(), tokenDecimals, quoteTokenType, nil, inviteBonusAmt)
	}
	return inviteRelations
}

func innerSettleUserFee(db vm_db.VmDb, reader util.ConsensusReader, address []byte, tokenDecimals, quoteTokenType int32, baseFee, inviteBonusFee []byte) {
	periodId := GetCurrentPeriodId(db, reader)
	userFees, _ := GetUserFees(db, address)
	feeLen := len(userFees.Fees)

	if feeLen > 0 && periodId == userFees.Fees[feeLen-1].Period {
		var foundToken = false
		for _, feeAcc := range userFees.Fees[feeLen-1].UserFees {
			if feeAcc.QuoteTokenType == quoteTokenType {
				if baseFee != nil {
					feeAcc.BaseAmount = AddBigInt(feeAcc.BaseAmount, AdjustAmountToQuoteTokenType(baseFee, tokenDecimals, quoteTokenType).Bytes())
				}
				if inviteBonusFee != nil {
					feeAcc.InviteBonusAmount = AddBigInt(feeAcc.InviteBonusAmount, AdjustAmountToQuoteTokenType(inviteBonusFee, tokenDecimals, quoteTokenType).Bytes())
				}
				foundToken = true
				break
			}
		}
		if !foundToken {
			userFees.Fees[feeLen-1].UserFees = append(userFees.Fees[feeLen-1].UserFees, newFeeAccount(tokenDecimals, quoteTokenType, baseFee, inviteBonusFee))
		}
	} else {
		userFeeByPeriodId := &dexproto.UserFeeWithPeriod{}
		userFeeByPeriodId.Period = periodId
		userFeeByPeriodId.UserFees = []*dexproto.UserFeeAccount{newFeeAccount(tokenDecimals, quoteTokenType, baseFee, inviteBonusFee)}
		userFees.Fees = append(userFees.Fees, userFeeByPeriodId)
	}
	SaveUserFees(db, address, userFees)
}

func SettleBrokerFeeSum(db vm_db.VmDb, reader util.ConsensusReader, feeActions []*dexproto.UserFeeSettle, marketInfo *MarketInfo) {
	var (
		incAmt               []byte
		brokerFeeSumByPeriod *BrokerFeeSumByPeriod
	)
	for _, feeAction := range feeActions {
		incAmt = AddBigInt(incAmt, feeAction.BrokerFee)
	}

	brokerFeeSumByPeriod, _ = GetCurrentBrokerFeeSum(db, reader, marketInfo.Owner)
	var foundToken bool
	for _, brokerFeeSum := range brokerFeeSumByPeriod.BrokerFees {
		if bytes.Equal(marketInfo.QuoteToken, brokerFeeSum.Token) {
			var foundMarket bool
			for _, mkFee := range brokerFeeSum.MarketFees {
				if mkFee.MarketId == marketInfo.MarketId {
					incBrokerMarketFee(marketInfo, mkFee, incAmt)
					foundMarket = true
					break
				}
			}
			if !foundMarket {
				brokerFeeSum.MarketFees = append(brokerFeeSum.MarketFees, newBrokerMarketFee(marketInfo, incAmt))
			}
			foundToken = true
			break
		}
	}
	if !foundToken {
		brokerFeeAcc := &dexproto.BrokerFeeAccount{}
		brokerFeeAcc.MarketFees = append(brokerFeeAcc.MarketFees, newBrokerMarketFee(marketInfo, incAmt))
		brokerFeeSumByPeriod.BrokerFees = append(brokerFeeSumByPeriod.BrokerFees, brokerFeeAcc)
	}

	SaveCurrentBrokerFeeSum(db, reader, marketInfo.Owner, brokerFeeSumByPeriod)
}

func OnDepositVx(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, depositAmount *big.Int, updatedVxAccount *dexproto.Account) {
	doSettleVxFunds(db, reader, address.Bytes(), depositAmount, updatedVxAccount)
}

func OnWithdrawVx(db vm_db.VmDb, reader util.ConsensusReader, address types.Address, withdrawAmount *big.Int, updatedVxAccount *dexproto.Account) {
	doSettleVxFunds(db, reader, address.Bytes(), new(big.Int).Neg(withdrawAmount), updatedVxAccount)
}

func OnSettleVx(db vm_db.VmDb, reader util.ConsensusReader, address []byte, fundSettle *dexproto.FundSettle, updatedVxAccount *dexproto.Account) {
	amtChange := SubBigInt(fundSettle.IncAvailable, fundSettle.ReduceLocked)
	doSettleVxFunds(db, reader, address, amtChange, updatedVxAccount)
}

func rollFeeSum(db vm_db.VmDb, reader util.ConsensusReader) (rolledFeeSumByPeriod *FeeSumByPeriod) {
	formerId := GetFeeSumLastPeriodIdForRoll(db)
	rolledFeeSumByPeriod = &FeeSumByPeriod{}
	if formerId > 0 {
		if formerFeeSumByPeriod, ok := GetFeeSumByPeriodId(db, formerId); !ok {
			panic(NoFeeSumFoundForValidPeriodErr)
		} else {
			rolledFeeSumByPeriod.LastValidPeriod = formerId
			for _, feeForDividend := range formerFeeSumByPeriod.FeesForDividend {
				rolledFee := &dexproto.FeeSumForDividend{}
				rolledFee.Token = feeForDividend.Token
				_, rolledAmount := splitDividendPool(feeForDividend)
				rolledFee.DividendPoolAmount = rolledAmount.Bytes()
				rolledFeeSumByPeriod.FeesForDividend = append(rolledFeeSumByPeriod.FeesForDividend, rolledFee)
			}
		}
	}
	SaveFeeSumLastPeriodIdForRoll(db, reader)
	return
}

func splitDividendPool(feeSumAcc *dexproto.FeeSumForDividend) (toDividendAmt, rolledAmount *big.Int) {
	dividendPoolTotal := AddBigInt(feeSumAcc.DividendPoolAmount, feeSumAcc.BaseAmount)
	toDividendAmt = new(big.Int).SetBytes(CalculateAmountForRate(dividendPoolTotal, PerPeriodDividendRate)) // %1
	rolledAmount = new(big.Int).Sub(new(big.Int).SetBytes(dividendPoolTotal), toDividendAmt)                // 99%
	return
}

// only settle validAmount and amount changed from previous period
func doSettleVxFunds(db vm_db.VmDb, reader util.ConsensusReader, addressBytes []byte, amtChange *big.Int, updatedVxAccount *dexproto.Account) {
	var (
		vxFunds               *VxFunds
		userNewAmt, sumChange *big.Int
		periodId              uint64
		fundsLen              int
		needUpdate            bool
	)
	vxFunds, _ = GetVxFundsFrom(db, addressBytes)
	periodId = GetCurrentPeriodId(db, reader)
	fundsLen = len(vxFunds.Funds)
	userNewAmt = new(big.Int).SetBytes(AddBigInt(updatedVxAccount.Available, updatedVxAccount.Locked))
	if fundsLen == 0 { //need append new period
		if IsValidVxAmountForDividend(userNewAmt) {
			fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
			sumChange = userNewAmt
			needUpdate = true
		}
	} else if vxFunds.Funds[fundsLen-1].Period == periodId { //update current period
		if IsValidVxAmountForDividend(userNewAmt) {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = userNewAmt
			}
			vxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
		} else {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = NegativeAmount(vxFunds.Funds[fundsLen-1].Amount)
			}
			if fundsLen > 1 { // in case fundsLen > 1, update last period to diff the condition of current period not changed ever from last saved period
				vxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
			} else { // clear funds in case only current period saved and not valid any more
				vxFunds.Funds = nil
			}
		}
		needUpdate = true
	} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
		if IsValidVxAmountForDividend(userNewAmt) {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = amtChange
			} else {
				sumChange = userNewAmt
			}
			fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
			needUpdate = true
		} else {
			if IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
				sumChange = NegativeAmount(vxFunds.Funds[fundsLen-1].Amount)
				fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
				vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
				needUpdate = true
			}
		}
	}

	if len(vxFunds.Funds) > 0 && needUpdate {
		SaveVxFunds(db, addressBytes, vxFunds)
	} else if len(vxFunds.Funds) == 0 && fundsLen > 0 {
		DeleteVxFunds(db, addressBytes)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		vxSumFunds, _ := GetVxSumFundsFromDb(db)
		sumFundsLen := len(vxSumFunds.Funds)
		if sumFundsLen == 0 {
			if sumChange.Sign() > 0 {
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Period: periodId, Amount: sumChange.Bytes()})
			} else {
				panic(fmt.Errorf("vxFundSum initiation get negative value"))
			}
		} else {
			sumRes := new(big.Int).Add(new(big.Int).SetBytes(vxSumFunds.Funds[sumFundsLen-1].Amount), sumChange)
			if sumRes.Sign() < 0 {
				panic(fmt.Errorf("vxFundSum updated res get negative value"))
			}
			if vxSumFunds.Funds[sumFundsLen-1].Period == periodId {
				vxSumFunds.Funds[sumFundsLen-1].Amount = sumRes.Bytes()
			} else {
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Amount: sumRes.Bytes(), Period: periodId})
			}
		}
		SaveVxSumFundsToDb(db, vxSumFunds)
	}
}

func getInviteBonusInfo(db vm_db.VmDb, addr []byte, inviteRelations *map[types.Address]*types.Address, fee []byte) (bool, *types.Address, []byte) {
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
			return true, inviter, CalculateAmountForRate(fee, InviteBonusRate)
		} else {
			return false, nil, nil
		}
	}
}

func newFeeAccount(tokenDecimals, quoteTokenType int32, baseAmount, inviteBonusAmount []byte) *dexproto.UserFeeAccount {
	account := &dexproto.UserFeeAccount{}
	account.QuoteTokenType = quoteTokenType
	account.BaseAmount = AdjustAmountToQuoteTokenType(baseAmount, tokenDecimals, quoteTokenType).Bytes()
	account.InviteBonusAmount = AdjustAmountToQuoteTokenType(inviteBonusAmount, tokenDecimals, quoteTokenType).Bytes()
	return account
}

func newFeeSumForDividend(token, baseAmount, dividendPoolAmount []byte) *dexproto.FeeSumForDividend {
	account := &dexproto.FeeSumForDividend{}
	account.Token = token
	account.BaseAmount = baseAmount
	account.DividendPoolAmount = dividendPoolAmount
	return account
}

func newFeeSumForMine(quoteTokenType int32, baseAmount, inviteBonusAmount []byte) *dexproto.FeeSumForMine {
	account := &dexproto.FeeSumForMine{}
	account.QuoteTokenType = quoteTokenType
	account.BaseAmount = baseAmount
	account.InviteBonusAmount = inviteBonusAmount
	return account
}

func newBrokerMarketFee(marketInfo *MarketInfo, amount []byte) *dexproto.BrokerMarketFee {
	account := &dexproto.BrokerMarketFee{}
	account.MarketId = marketInfo.MarketId
	account.TakerBrokerFeeRate = marketInfo.TakerBrokerFeeRate
	account.MakerBrokerFeeRate = marketInfo.MakerBrokerFeeRate
	account.Amount = amount
	return account
}

func incBrokerMarketFee(marketInfo *MarketInfo, marketFee *dexproto.BrokerMarketFee, incAmt []byte) {
	marketFee.TakerBrokerFeeRate = marketInfo.TakerBrokerFeeRate
	marketFee.MakerBrokerFeeRate = marketInfo.MakerBrokerFeeRate
	marketFee.Amount = AddBigInt(marketFee.Amount, incAmt)
}
