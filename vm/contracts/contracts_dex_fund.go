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
		{"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]}
		{"type":"function","name":"DexFundDividend", "inputs":[{"name":"periodId","type":"uint32"}]}
	]`

	MethodNameDexFundUserDeposit  = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder     = "DexFundNewOrder"
	MethodNameDexFundSettleOrders = "DexFundSettleOrders"
	MethodNameDexFundDividend     = "DexFundDividend"
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

func (md *MethodDexFundUserDeposit) GetQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundDepositGas, data)
}

func (md *MethodDexFundUserDeposit) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 {
		return fmt.Errorf("deposit amount is zero")
	}
	if err, _ := dex.GetTokenInfo(db, block.TokenId); err != nil {
		return err
	}
	return nil
}

func (md *MethodDexFundUserDeposit) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		dexFund = &dex.UserFund{}
		err     error
	)
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	walletAvailable := db.GetBalance(&sendBlock.AccountAddress, &sendBlock.TokenId)
	if walletAvailable.Cmp(sendBlock.Amount) < 0 {
		return []*SendBlock{}, fmt.Errorf("deposit amount exceed token balance")
	}
	account, exists := dex.GetAccountByTokeIdFromFund(dexFund, sendBlock.TokenId)
	available := new(big.Int).SetBytes(account.Available)
	account.Available = available.Add(available, sendBlock.Amount).Bytes()
	if !exists {
		dexFund.Accounts = append(dexFund.Accounts, account)
	}
	// must do after account update by deposit
	if bytes.Equal(sendBlock.TokenId.Bytes(), dex.VxTokenBytes) {
		if err = onDepositVx(db, db.GetSnapshotBlockByHash(&block.SnapshotHash).Height, sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
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

func (md *MethodDexFundUserWithdraw) GetQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundWithdrawGas, data)
}

func (md *MethodDexFundUserWithdraw) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundWithDraw)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundUserWithdraw, block.Data); err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 {
		return fmt.Errorf("withdraw amount is zero")
	}
	if tokenInfo := cabi.GetTokenById(db, param.Token); tokenInfo == nil {
		return fmt.Errorf("token to withdraw is invalid")
	}
	return nil
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
		if err = onWithdrawVx(db, db.GetSnapshotBlockByHash(&block.SnapshotHash).Height, sendBlock.AccountAddress, param.Amount, account); err != nil {
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

func (md *MethodDexFundNewOrder) GetQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewOrderGas, data)
}

func (md *MethodDexFundNewOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, block.Data); err != nil {
		return err
	}
	if err = dex.CheckOrderParam(db, param); err != nil {
		return err
	}
	return nil
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
	dex.RenderOrder(order, param, db, sendBlock.AccountAddress, db.GetSnapshotBlockByHash(&block.SnapshotHash).Timestamp)
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	if _, err = tryLockFundForNewOrder(db, dexFund, order); err != nil {
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

func (md *MethodDexFundSettleOrders) GetQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewOrderGas, data)
}

func (md *MethodDexFundSettleOrders) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	if !bytes.Equal(block.AccountAddress.Bytes(), types.AddressDexTrade.Bytes()) {
		return fmt.Errorf("invalid block source")
	}
	param := new(dex.ParamDexSerializedData)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSettleOrders, block.Data); err != nil {
		return err
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return err
	}
	if err = dex.CheckSettleActions(settleActions); err != nil {
		return err
	}
	return nil
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
	for _, fundAction := range settleActions.FundActions {
		if err = doSettleFund(db, fundAction, db.GetSnapshotBlockByHash(&block.SnapshotHash).Height); err != nil {
			return []*SendBlock{}, err
		}
	}
	for _, feeAction := range settleActions.FeeActions {
		if err = doSettleFee(db, db.GetSnapshotBlockByHash(&block.SnapshotHash).Height, feeAction); err != nil {
			return []*SendBlock{}, err
		}
	}
	return []*SendBlock{}, nil
}

type MethodDexVxDividend struct {
}

func (md *MethodDexVxDividend) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexVxDividend) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexVxDividend) GetQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundVxDividendGas, data)
}

func (md *MethodDexVxDividend) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var (
		vxFundSum *dex.VxFunds
		err       error
	)
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundDividend, block.Data); err != nil {
		return err
	}
	vxFundSum, err = dex.GetVxFundSumKFromStorage(db)
	sumLen := len(vxFundSum.Funds)
	if sumLen == 0 {
		return fmt.Errorf("no fee for dividend")
	} else if vxFundSum.Funds[sumLen-1].Period != param.PeriodId {
		return fmt.Errorf("period id not valid")
	}
	return err
}

func (md MethodDexVxDividend) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	return []*SendBlock{}, nil
}

func tryLockFundForNewOrder(db vmctxt_interface.VmDatabase, dexFund *dex.UserFund, order *dexproto.Order) (needUpdate bool, err error) {
	return checkAndLockFundForNewOrder(db, dexFund, order, false)
}

func checkAndLockFundForNewOrder(db vmctxt_interface.VmDatabase, dexFund *dex.UserFund, order *dexproto.Order, onlyCheck bool) (needUpdate bool, err error) {
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

func doSettleFund(db vmctxt_interface.VmDatabase, action *dexproto.FundSettle, snapshotBlockHeight uint64) error {
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
				if err = onSettleVx(db, snapshotBlockHeight, action, account); err != nil {
					return err
				}
			}
			//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
		}
		return err
	}
	return nil
}

func doSettleFee(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64, feeAction *dexproto.FeeSettle) error {
	var (
		dexFee *dex.Fee
		err    error
	)
	if dexFee, err = dex.GetFeeFromStorage(storage, snapshotBlockHeight); err != nil {
		return err
	} else {
		if dexFee.Divided {
			return fmt.Errorf("err status for settle fee as fee already divided for height %d", snapshotBlockHeight)
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
	if err = dex.SaveFeeToStorage(storage, snapshotBlockHeight, dexFee); err != nil {
		return err
	} else {
		return nil
	}
}

func onDepositVx(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64, address types.Address, depositAmount *big.Int, updatedAmount *dexproto.Account) error {
	return doSettleVxFunds(storage, snapshotBlockHeight, address.Bytes(), depositAmount, updatedAmount)
}

func onWithdrawVx(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64, address types.Address, withdrawAmount *big.Int, updatedAmount *dexproto.Account) error {
	amtChanged := new(big.Int).Sub(big.NewInt(0), withdrawAmount)
	return doSettleVxFunds(storage, snapshotBlockHeight, address.Bytes(), amtChanged, updatedAmount)
}

func onSettleVx(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64, action *dexproto.FundSettle, updatedAmount *dexproto.Account) error {
	amtChange := dex.SubBigInt(action.IncAvailable, action.ReduceLocked)
	return doSettleVxFunds(storage, snapshotBlockHeight, action.Address, amtChange, updatedAmount)
}

// only settle validAmount and amount changed from previous period
func doSettleVxFunds(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64, addressBytes []byte, amtChange *big.Int, updatedAccount *dexproto.Account) error {
	var (
		dexVxFunds            *dex.VxFunds
		userNewAmt, sumChange *big.Int
		err                   error
		periodId              uint64
		fundsLen              int
	)
	if dexVxFunds, err = dex.GetVxFundsFromStorage(storage, addressBytes); err != nil {
		return err
	} else {
		periodId = dex.GetPeriodIdFromHeight(snapshotBlockHeight)
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
		if err = dex.SaveVxFundsToStorage(storage, addressBytes, dexVxFunds); err != nil {
			return err
		}
	} else if fundsLen > 0 {
		dex.DeleteVxFundsToStorage(storage, addressBytes)
	}

	if sumChange != nil && dex.CmpToBigZero(sumChange.Bytes()) != 0 {
		var dexVxFundSum *dex.VxFunds
		if dexVxFundSum, err = dex.GetVxFundSumKFromStorage(storage); err != nil {
			return err
		} else {
			fundSumLen := len(dexVxFundSum.Funds)
			if fundSumLen == 0 && sumChange.Sign() > 0 {
				dexVxFundSum.Funds = append(dexVxFundSum.Funds, &dexproto.VxFundWithPeriod{Amount: sumChange.Bytes(), Period: periodId})
			} else {
				sumRes := new(big.Int).Add(new(big.Int).SetBytes(dexVxFundSum.Funds[fundSumLen-1].Amount), sumChange)
				if sumRes.Sign() < 0 {
					return fmt.Errorf("vxFundSum get negative value")
				}
				if dexVxFundSum.Funds[fundSumLen-1].Period == periodId {
					dexVxFundSum.Funds[fundSumLen-1].Amount = sumRes.Bytes()
				} else {
					dexVxFundSum.Funds = append(dexVxFundSum.Funds, &dexproto.VxFundWithPeriod{Amount: sumRes.Bytes(), Period: periodId})
				}
			}
			dex.SaveVxFundSumKToStorage(storage, dexVxFundSum)
		}
	}
	return nil
}
