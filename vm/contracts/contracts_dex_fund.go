package contracts

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strings"
)

const (
	jsonDexFund = `
	[
		{"type":"function","name":"DexFundUserDeposit", "inputs":[]},
		{"type":"function","name":"DexFundUserWithdraw", "inputs":[{"name":"token","type":"tokenId"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"orderId","type":"bytes"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}, {"name":"orderType", "type":"uint32"}, {"name":"price", "type":"string"}, {"name":"quantity", "type":"uint256"}]},
		{"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexFundFeeDividend", "inputs":[{"name":"periodId","type":"uint32"}]}
	]`

	MethodNameDexFundUserDeposit  = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder     = "DexFundNewOrder"
	MethodNameDexFundSettleOrders = "DexFundSettleOrders"
	MethodNameDexFundFeeDividend  = "DexFundFeeDividend"
)

var (
	ABIDexFund, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
)

type MethodDexFundUserDeposit struct {
}

func (md *MethodDexFundUserDeposit) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserDeposit) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserDeposit) GetQuota() uint64 {
	return 1000
}

func (md *MethodDexFundUserDeposit) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, dexFundDepositGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() <= 0 {
		return quotaLeft, fmt.Errorf("deposit amount is zero")
	}
	if err, _ = dex.GetTokenInfo(db, block.TokenId); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundUserDeposit) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		dexFund = &dex.UserFund{}
		err     error
	)
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	account, exists := dex.GetAccountByTokeIdFromFund(dexFund, sendBlock.TokenId)
	available := new(big.Int).SetBytes(account.Available)
	account.Available = available.Add(available, sendBlock.Amount).Bytes()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	// must do after account update by deposit
	if bytes.Equal(sendBlock.TokenId.Bytes(), dex.VxTokenBytes) {
		if err = onDepositVx(db, sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
			return []*SendBlock{}, err
		}
	}
	return []*SendBlock{}, dex.SaveUserFundToStorage(db, sendBlock.AccountAddress, dexFund)
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserWithdraw) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserWithdraw) GetQuota() uint64 {
	return 1000
}

func (md *MethodDexFundUserWithdraw) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundWithdrawGas); err != nil {
		return quotaLeft, err
	}
	param := new(dex.ParamDexFundWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, block.Data); err != nil {
		return quotaLeft, err
	}
	if param.Amount.Sign() <= 0 {
		return quotaLeft, fmt.Errorf("withdraw amount is zero")
	}
	if tokenInfo := cabi.GetTokenById(db, param.Token); tokenInfo == nil {
		return quotaLeft, fmt.Errorf("token to withdraw is invalid")
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundUserWithdraw) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(dex.ParamDexFundWithDraw)
	var (
		dexFund = &dex.UserFund{}
		err     error
	)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	account, _ := dex.GetAccountByTokeIdFromFund(dexFund, param.Token)
	available := big.NewInt(0).SetBytes(account.Available)
	if available.Cmp(param.Amount) < 0 {
		return []*SendBlock{}, fmt.Errorf("withdraw amount exceed fund available")
	}
	available = available.Sub(available, param.Amount)
	account.Available = available.Bytes()
	// must do after account update by withdraw
	if bytes.Equal(param.Token.Bytes(), dex.VxTokenBytes) {
		if err = onWithdrawVx(db, sendBlock.AccountAddress, param.Amount, account); err != nil {
			return []*SendBlock{}, err
		}
	}
	if err = dex.SaveUserFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return []*SendBlock{}, err
	}
	return []*SendBlock{
		{
			block,
			sendBlock.AccountAddress,
			ledger.BlockTypeSendCall,
			param.Amount,
			param.Token,
			[]byte{},
		},
	}, nil
}

type MethodDexFundNewOrder struct {
}

func (md *MethodDexFundNewOrder) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewOrder) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundNewOrder) GetQuota() uint64 {
	return 1000
}

func (md *MethodDexFundNewOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundNewOrderGas); err != nil {
		return quotaLeft, err
	}
	param := new(dex.ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, block.Data); err != nil {
		return quotaLeft, err
	}
	if err = dex.CheckOrderParam(db, param); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md *MethodDexFundNewOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		dexFund        = &dex.UserFund{}
		tradeBlockData []byte
		err            error
		orderBytes     []byte
	)
	param := new(dex.ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	order := &dexproto.Order{}
	dex.RenderOrder(order, param, db, sendBlock.AccountAddress, db.CurrentSnapshotBlock().Timestamp)
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	if _, err = tryLockFundForNewOrder(dexFund, order); err != nil {
		return []*SendBlock{}, err
	}
	if err = dex.SaveUserFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return []*SendBlock{}, err
	}
	if orderBytes, err = proto.Marshal(order); err != nil {
		return []*SendBlock{}, err
	}
	if tradeBlockData, err = ABIDexTrade.PackMethod(MethodNameDexTradeNewOrder, orderBytes); err != nil {
		return []*SendBlock{}, err
	}
	return []*SendBlock{
		{
			block,
			types.AddressDexTrade,
			ledger.BlockTypeSendCall,
			big.NewInt(0),
			ledger.ViteTokenId, // no need send token
			tradeBlockData,
		},
	}, nil
}

type MethodDexFundSettleOrders struct {
}

func (md *MethodDexFundSettleOrders) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleOrders) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundSettleOrders) GetQuota() uint64 {
	return 1000
}

func (md *MethodDexFundSettleOrders) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var err error
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundSettleOrdersGas); err != nil {
		return quotaLeft, err
	}
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexTrade.Bytes()) {
		return quotaLeft, fmt.Errorf("invalid block source")
	}
	param := new(dex.ParamDexSerializedData)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, block.Data); err != nil {
		return quotaLeft, err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return quotaLeft, err
	}
	if err = dex.CheckSettleActions(settleActions); err != nil {
		return quotaLeft, err
	}
	return util.UseQuotaForData(block.Data, quotaLeft)
}

func (md MethodDexFundSettleOrders) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	if !bytes.Equal(sendBlock.AccountAddress.Bytes(), types.AddressDexTrade.Bytes()) {
		return []*SendBlock{}, fmt.Errorf("invalid block source")
	}
	param := new(dex.ParamDexSerializedData)
	var err error
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return []*SendBlock{}, err
	}
	//TODO merge actions for one address
	for _, fundAction := range settleActions.FundActions {
		if err = doSettleFund(db, fundAction); err != nil {
			return []*SendBlock{}, err
		}
	}
	for _, feeAction := range settleActions.FeeActions {
		if err = doSettleFee(db, feeAction); err != nil {
			return []*SendBlock{}, err
		}
	}
	return []*SendBlock{}, nil
}

type MethodDexFundFeeDividend struct {
}

func (md *MethodDexFundFeeDividend) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundFeeDividend) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundFeeDividend) GetQuota() uint64 {
	return 1000
}

func (md *MethodDexFundFeeDividend) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	var (
		err        error
	)
	if quotaLeft, err = util.UseQuota(quotaLeft, dexFundVxDividendGas); err != nil {
		return quotaLeft, err
	}
	//TODO check periodId is a finished period, not current period
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundFeeDividend, block.Data); err != nil {
		return quotaLeft, err
	}
	if lastDividendId := dex.GetLastDividendIdFromStorage(db); lastDividendId > 0 && param.PeriodId != lastDividendId + 1 {
		return quotaLeft, fmt.Errorf("dividend period id not equals to expected id %d", lastDividendId + 1)
	}
	return quotaLeft, err
}

func (md MethodDexFundFeeDividend) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		err        error
	)
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundFeeDividend, block.Data); err != nil {
		return []*SendBlock{}, err
	}
	if lastDividendId := dex.GetLastDividendIdFromStorage(db); lastDividendId > 0 && param.PeriodId != lastDividendId + 1 {
		return []*SendBlock{}, fmt.Errorf("dividend period id not equals to expected id %d", lastDividendId + 1)
	}
	if err = doDivideFees(db, param.PeriodId); err != nil {
		return []*SendBlock{}, err
	} else {
		dex.SaveLastDividendIdToStorage(db, param.PeriodId)
	}
	return []*SendBlock{}, nil
}

func tryLockFundForNewOrder(dexFund *dex.UserFund, order *dexproto.Order) (needUpdate bool, err error) {
	return checkAndLockFundForNewOrder(dexFund, order, false)
}

func checkAndLockFundForNewOrder(dexFund *dex.UserFund, order *dexproto.Order, onlyCheck bool) (needUpdate bool, err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId           *types.TokenTypeId
		lockAmountToInc       *big.Int
	)
	switch order.Side {
	case false: //buy
		lockToken = order.QuoteToken
		if order.Type == dex.Limited {
			lockAmount = dex.AddBigInt(order.Amount, order.LockedBuyFee)
		}
	case true: // sell
		lockToken = order.TradeToken
		lockAmount = order.Quantity
	}
	if lockTokenId, err = dex.FromBytesToTokenTypeId(lockToken); err != nil {
		return false, err
	}
	//var tokenName string
	//if tokenInfo := cabi.GetTokenById(db, *lockTokenId); tokenInfo != nil {
	//	tokenName = tokenInfo.TokenName
	//}
	account, exists := dex.GetAccountByTokeIdFromFund(dexFund, *lockTokenId)
	available := big.NewInt(0).SetBytes(account.Available)
	if order.Type != dex.Market || order.Side { // limited or sell order
		lockAmountToInc = new(big.Int).SetBytes(lockAmount)
		//fmt.Printf("token %s, available %s , lockAmountToInc %s\n", tokenName, available.String(), lockAmountToInc.String())
		if available.Cmp(lockAmountToInc) < 0 {
			return false, fmt.Errorf("order lock amount exceed fund available")
		}
	}

	if onlyCheck {
		return false, nil
	}
	if !order.Side && order.Type == dex.Market { // buy or market order
		if available.Sign() <= 0 {
			return false, fmt.Errorf("no quote amount available for market sell order")
		} else {
			lockAmount = available.Bytes()
			//NOTE: use amount available for order amount to full fill
			order.Amount = lockAmount
			lockAmountToInc = available
			needUpdate = true
		}
	}
	available = available.Sub(available, lockAmountToInc)
	lockedInBig := new(big.Int).SetBytes(account.Locked)
	lockedInBig = lockedInBig.Add(lockedInBig, lockAmountToInc)
	account.Available = available.Bytes()
	account.Locked = lockedInBig.Bytes()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	return needUpdate, nil
}

func doSettleFund(db vmctxt_interface.VmDatabase, action *dexproto.FundSettle) error {
	address := &types.Address{}
	address.SetBytes([]byte(action.Address))
	if dexFund, err := dex.GetUserFundFromStorage(db, *address); err != nil {
		return err
	} else {
		if tokenId, err := dex.FromBytesToTokenTypeId(action.Token); err != nil {
			return err
		} else {
			if err, _ = dex.GetTokenInfo(db, *tokenId); err != nil {
				return err
			}
			account, exists := dex.GetAccountByTokeIdFromFund(dexFund, *tokenId)
			//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
			if dex.CmpToBigZero(action.ReduceLocked) > 0 {
				if dex.CmpForBigInt(action.ReduceLocked, account.Locked) > 0 {
					return fmt.Errorf("try reduce locked amount execeed locked")
				}
				account.Locked = dex.SubBigIntAbs(account.Locked, action.ReduceLocked)
			}
			if dex.CmpToBigZero(action.ReleaseLocked) > 0 {
				if dex.CmpForBigInt(action.ReleaseLocked, account.Locked) > 0 {
					return fmt.Errorf("try release locked amount execeed locked")
				}
				account.Locked = dex.SubBigIntAbs(account.Locked, action.ReleaseLocked)
				account.Available = dex.AddBigInt(account.Available, action.ReleaseLocked)
			}
			if dex.CmpToBigZero(action.IncAvailable) > 0 {
				account.Available = dex.AddBigInt(account.Available, action.IncAvailable)
			}
			if !exists {
				dexFund.Accounts = append(dexFund.Accounts, account)
			}
			if err = dex.SaveUserFundToStorage(db, *address, dexFund); err != nil {
				return err
			}
			// must do after account update by withdraw
			if bytes.Equal(action.Token, dex.VxTokenBytes) {
				if err = onSettleVx(db, action, account); err != nil {
					return err
				}
			}
			//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
		}
		return err
	}
	return nil
}

func doSettleFee(db vmctxt_interface.VmDatabase, feeAction *dexproto.FeeSettle) error {
	var (
		dexFee *dex.Fee
		err    error
	)
	if dexFee, err = dex.GetCurrentFeeFromStorage(db); err != nil {
		return err
	} else {
		if dexFee == nil { // need roll period when
			if formerPeriodId, err := rollFeePeriodId(db); err != nil {
				return err
			} else {
				dexFee = &dex.Fee{}
				dexFee.LastValidPeriod = formerPeriodId
			}
		}
		var foundToken = false
		for _, feeAcc := range dexFee.Fees {
			if bytes.Equal(feeAcc.Token, feeAction.Token) {
				feeAcc.Amount = dex.AddBigInt(feeAcc.Amount, feeAction.Amount)
				foundToken = true
				break
			}
		}
		if !foundToken {
			feeAcc := &dexproto.FeeAccount{}
			feeAcc.Token = feeAction.Token
			feeAcc.Amount = feeAction.Amount
			dexFee.Fees = append(dexFee.Fees, feeAcc)
		}
	}
	if err = dex.SaveCurrentFeeToStorage(db, dexFee); err != nil {
		return err
	} else {
		return nil
	}
}

func rollFeePeriodId(db vmctxt_interface.VmDatabase) (uint64, error) {
	formerId := dex.GetFeeLastPeriodIdForRoll(db)
	if err := dex.SaveFeeLastPeriodIdForRoll(db); err != nil {
		return 0, err
	} else {
		return formerId, err
	}
}

func onDepositVx(db vmctxt_interface.VmDatabase, address types.Address, depositAmount *big.Int, updatedAmount *dexproto.Account) error {
	return doSettleVxFunds(db, address.Bytes(), depositAmount, updatedAmount)
}

func onWithdrawVx(db vmctxt_interface.VmDatabase, address types.Address, withdrawAmount *big.Int, updatedAmount *dexproto.Account) error {
	amtChanged := new(big.Int).Sub(big.NewInt(0), withdrawAmount)
	return doSettleVxFunds(db, address.Bytes(), amtChanged, updatedAmount)
}

func onSettleVx(db vmctxt_interface.VmDatabase, action *dexproto.FundSettle, updatedAmount *dexproto.Account) error {
	amtChange := dex.SubBigInt(action.IncAvailable, action.ReduceLocked)
	return doSettleVxFunds(db, action.Address, amtChange, updatedAmount)
}

// only settle validAmount and amount changed from previous period
func doSettleVxFunds(db vmctxt_interface.VmDatabase, addressBytes []byte, amtChange *big.Int, updatedAccount *dexproto.Account) error {
	var (
		dexVxFunds            *dex.VxFunds
		userNewAmt, sumChange *big.Int
		err                   error
		periodId              uint64
		fundsLen              int
	)
	if dexVxFunds, err = dex.GetVxFundsFromStorage(db, addressBytes); err != nil {
		return err
	} else {
		if periodId, err = dex.GetCurrentPeriodIdFromStorage(db); err != nil {
			return err
		}
		fundsLen = len(dexVxFunds.Funds)
		userNewAmt = new(big.Int).SetBytes(dex.AddBigInt(updatedAccount.Available, updatedAccount.Locked))
		if fundsLen == 0 { //need append new period
			if dex.IsValidVxAmountForDividend(userNewAmt) {
				fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
				dexVxFunds.Funds = append(dexVxFunds.Funds, fundWithPeriod)
				sumChange = userNewAmt
			}
		} else if dexVxFunds.Funds[fundsLen-1].Period == periodId { //update current period
			if dex.IsValidVxAmountForDividend(userNewAmt) {
				if !dex.IsValidVxAmountBytesForDividend(dexVxFunds.Funds[fundsLen-1].Amount) {
					sumChange = userNewAmt
				} else {
					sumChange = amtChange
				}
				dexVxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
			} else {
				if dex.IsValidVxAmountBytesForDividend(dexVxFunds.Funds[fundsLen-1].Amount) {
					sumChange = dex.NegativeAmount(dexVxFunds.Funds[fundsLen-1].Amount)
				}
				if fundsLen > 1 { // in case fundsLen > 1, update last period to diff the condition of current period not changed ever from last saved period
					dexVxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
				} else { // clear funds in case only current period saved and not valid any more
					dexVxFunds.Funds = nil
				}
			}
		} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
			if dex.IsValidVxAmountForDividend(userNewAmt) {
				if !dex.IsValidVxAmountBytesForDividend(dexVxFunds.Funds[fundsLen-1].Amount) {
					sumChange = userNewAmt
				} else {
					sumChange = amtChange
				}
			} else {
				if dex.IsValidVxAmountBytesForDividend(dexVxFunds.Funds[fundsLen-1].Amount) {
					sumChange = dex.NegativeAmount(dexVxFunds.Funds[fundsLen-1].Amount)
				}
			}
			fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
			dexVxFunds.Funds = append(dexVxFunds.Funds, fundWithPeriod)
		}
	}
	if len(dexVxFunds.Funds) > 0 {
		if err = dex.SaveVxFundsToStorage(db, addressBytes, dexVxFunds); err != nil {
			return err
		}
	} else if fundsLen > 0 {
		dex.DeleteVxFundsFromStorage(db, addressBytes)
	}

	if sumChange != nil && sumChange.Sign() != 0 {
		var vxSumFunds *dex.VxFunds
		if vxSumFunds, err = dex.GetVxSumFundsFromStorage(db); err != nil {
			return err
		} else {
			if vxSumFunds == nil {
				vxSumFunds = &dex.VxFunds{}
			}
			fundSumLen := len(vxSumFunds.Funds)
			if fundSumLen == 0 && sumChange.Sign() > 0 {
				vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Amount: sumChange.Bytes(), Period: periodId})
			} else {
				sumRes := new(big.Int).Add(new(big.Int).SetBytes(vxSumFunds.Funds[fundSumLen-1].Amount), sumChange)
				if sumRes.Sign() < 0 {
					return fmt.Errorf("vxFundSum get negative value")
				}
				if vxSumFunds.Funds[fundSumLen-1].Period == periodId {
					vxSumFunds.Funds[fundSumLen-1].Amount = sumRes.Bytes()
				} else {
					vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Amount: sumRes.Bytes(), Period: periodId})
				}
			}
			dex.SaveVxSumFundsToStorage(db, vxSumFunds)
		}
	}
	return nil
}

func doDivideFees(db vmctxt_interface.VmDatabase, periodId uint64) error {
	var (
		dexFees map[uint64]*dex.Fee
		vxSumFunds *dex.VxFunds
		err error
	)
	if dexFees, err = dex.GetNotDividedFeesByPeriodIdFromStorage(db, periodId); err != nil {
		return err
	} else if len(dexFees) == 0 {// no fee to divide
		return nil
	}
	if vxSumFunds, err = dex.GetVxSumFundsFromStorage(db); err != nil {
		return err
	}
	foundVxSumFunds, vxSumAmtBytes, needUpdateVxSum := dex.MatchVxFundsByPeriod(vxSumFunds, periodId)
	if !foundVxSumFunds { // not found vxSumFunds
		return nil
	}
	if needUpdateVxSum {
		if err := dex.SaveVxSumFundsToStorage(db, vxSumFunds); err != nil {
			return err
		}
	}
	vxSumAmt := new(big.Int).SetBytes(vxSumAmtBytes)
	// sum fees from multi period not divided
	feeSums := make(map[types.TokenTypeId]*big.Int)
	for feePeriodId, fee := range dexFees {
		for _, feeAccount := range fee.Fees {
			if tokenId, err := dex.FromBytesToTokenTypeId(feeAccount.Token); err != nil {
				return err
			} else {
				if amt, ok := feeSums[*tokenId]; !ok {
					feeSums[*tokenId] = new(big.Int).SetBytes(feeAccount.Amount)
				} else {
					feeSums[*tokenId] = amt.Add(amt, new(big.Int).SetBytes(feeAccount.Amount))
				}
			}
		}
		dex.DeleteFeeByPeriodIdFromStorage(db, feePeriodId)
	}

	var (
		usersVxFunds = make(map[types.Address]*dex.VxFunds)
		dividedVxAmt = big.NewInt(0)
	)

	iterator := db.NewStorageIterator(&types.AddressDexFund, dex.VxFundKeyPrefix)
	for {
		if userVxFundsKey, userVxFundsBytes, ok := iterator.Next(); !ok {
			break
		} else {
			addressBytes := userVxFundsKey[len(dex.VxFundKeyPrefix):]
			address := types.Address{}
			if err = address.SetBytes(addressBytes); err != nil {
				return err
			} else {
				userVxFunds := &dex.VxFunds{}
				if userVxFunds, err = userVxFunds.DeSerialize(userVxFundsBytes); err != nil {
					return err
				}
				usersVxFunds[address] = userVxFunds
			}
		}
	}

	if len(usersVxFunds) == 0 {
		return fmt.Errorf("not found user with vx funds in period %d, while vxSumFund is valid", periodId)
	}

	for address, userVxFunds := range usersVxFunds {
		var userFeeDividend= make(map[types.TokenTypeId]*big.Int)
		foundVxFunds, userVxAmtBytes, needUpdateVxFunds := dex.MatchVxFundsByPeriod(userVxFunds, periodId)
		if !foundVxFunds {
			continue
		}
		if needUpdateVxFunds {
			if err := dex.SaveVxFundsToStorage(db, address.Bytes(), userVxFunds); err != nil {
				return err
			}
		}
		userVxAmount := new(big.Int).SetBytes(userVxAmtBytes)
		dividedVxAmt.Add(dividedVxAmt, userVxAmount)
		userFeePortion := new(big.Float).Quo(new(big.Float).SetInt(userVxAmount), new(big.Float).SetInt(vxSumAmt))
		for tokenId, feeAmtToDivide := range feeSums {
			userFeeAmt := dex.RoundAmount(new(big.Float).Mul(new(big.Float).SetInt(feeAmtToDivide), userFeePortion))
			leaveFeeAmtSum := new(big.Int).Sub(feeAmtToDivide, userFeeAmt)
			if leaveFeeAmtSum.Sign() <= 0 || dividedVxAmt.Cmp(vxSumAmt) >= 0 {
				userFeeDividend[tokenId] = feeAmtToDivide
				delete(feeSums, tokenId)
			} else {
				userFeeDividend[tokenId] = userFeeAmt
				feeAmtToDivide = leaveFeeAmtSum
			}
		}
		if len(userFeeDividend) > 0 {
			if userFund, err := dex.GetUserFundFromStorage(db, address); err != nil {
				return err
			} else {
				for _, fund := range userFund.Accounts {
					if tk, err := dex.FromBytesToTokenTypeId(fund.Token); err != nil {
						return err
					} else {
						if amt, ok := userFeeDividend[*tk]; !ok {
							acc := &dexproto.Account{}
							acc.Token = tk.Bytes()
							acc.Available = amt.Bytes()
							userFund.Accounts = append(userFund.Accounts, acc)
						}
					}
				}
				if err := dex.SaveUserFundToStorage(db, address, userFund); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
