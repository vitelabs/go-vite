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
		{"type":"function","name":"DexFundFeeDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundMinedVxDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundNewMarket", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]}
	]`

	MethodNameDexFundUserDeposit     = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw    = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder        = "DexFundNewOrder"
	MethodNameDexFundSettleOrders    = "DexFundSettleOrders"
	MethodNameDexFundFeeDividend     = "DexFundFeeDividend"
	MethodNameDexFundMinedVxDividend = "DexFundMinedVxDividend"
	MethodNameDexFundNewMarket       = "DexFundNewMarket"
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

func (md *MethodDexFundUserDeposit) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundDepositGas, data)
}

func (md *MethodDexFundUserDeposit) GetReceiveQuota() uint64 {
	return dexFundDepositReceiveGas
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
	if account, err := depositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount); err != nil {
		return []*SendBlock{}, err
	} else {
		// must do after account updated by deposit
		if bytes.Equal(sendBlock.TokenId.Bytes(), dex.VxTokenBytes) {
			if err = onDepositVx(db, sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
				return []*SendBlock{}, err
			}
		}
		return []*SendBlock{}, nil
	}
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserWithdraw) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundUserWithdraw) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundWithdrawGas, data)
}

func (md *MethodDexFundUserWithdraw) GetReceiveQuota() uint64 {
	return dexFundWithdrawReceiveGas
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
		return handleReceiveErr(db, err)
	}
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return handleReceiveErr(db, err)
	}
	account, _ := dex.GetAccountByTokeIdFromFund(dexFund, param.Token)
	available := dex.SubBigInt(account.Available, param.Amount.Bytes())
	if available.Sign() < 0 {
		return handleReceiveErr(db, fmt.Errorf("withdraw amount exceed fund available"))
	}
	account.Available = available.Bytes()
	// must do after account updated by withdraw
	if bytes.Equal(param.Token.Bytes(), dex.VxTokenBytes) {
		if err = onWithdrawVx(db, sendBlock.AccountAddress, param.Amount, account); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	if err = dex.SaveUserFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return handleReceiveErr(db, err)
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

func (md *MethodDexFundNewOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewOrderGas, data)
}

func (md *MethodDexFundNewOrder) GetReceiveQuota() uint64 {
	return dexFundNewOrderReceiveGas
}

func (md *MethodDexFundNewOrder) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, block.Data); err != nil {
		return err
	}
	if err = dex.PreCheckOrderParam(db, param); err != nil {
		return err
	}
	return nil
}

func (md *MethodDexFundNewOrder) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		dexFund        = &dex.UserFund{}
		tradeBlockData []byte
		err            error
		orderInfoBytes []byte
	)
	param := new(dex.ParamDexFundNewOrder)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewOrder, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	orderInfo := &dexproto.OrderInfo{}
	if dexErr := dex.RenderOrder(orderInfo, param, db, sendBlock.AccountAddress); dexErr != nil {
		return handleNewOrderFail(db, orderInfo, dexErr.Code())
	}
	if dexFund, err = dex.GetUserFundFromStorage(db, sendBlock.AccountAddress); err != nil {
		return handleNewOrderFail(db, orderInfo, dex.NewOrderGetFundFail)
	}
	if _, err = checkAndLockFundForNewOrder(dexFund, orderInfo); err != nil {
		return handleNewOrderFail(db, orderInfo, dex.NewOrderLockFundFail)
	}
	if err = dex.SaveUserFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return handleNewOrderFail(db, orderInfo, dex.NewOrderSaveFundFail)
	}
	if orderInfoBytes, err = proto.Marshal(orderInfo); err != nil {
		return handleNewOrderFail(db, orderInfo, dex.NewOrderInternalErr)
	}
	if tradeBlockData, err = ABIDexTrade.PackMethod(MethodNameDexTradeNewOrder, orderInfoBytes); err != nil {
		return handleNewOrderFail(db, orderInfo, dex.NewOrderInternalErr)
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

func (md *MethodDexFundSettleOrders) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundSettleOrdersGas, data)
}

func (md *MethodDexFundSettleOrders) GetReceiveQuota() uint64 {
	return dexFundSettleOrdersReceiveGas
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
		return handleReceiveErr(db, err)
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return handleReceiveErr(db, err)
	}
	for _, fundAction := range settleActions.FundActions {
		if err = doSettleFund(db, fundAction); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	if len(settleActions.FeeActions) > 0 {
		if err = settleFeeSum(db, settleActions.FeeActions); err != nil {
			return handleReceiveErr(db, err)
		}
		for _, feeAction := range settleActions.FeeActions {
			if err = settleUserFees(db, feeAction); err != nil {
				return handleReceiveErr(db, err)
			}
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

func (md *MethodDexFundFeeDividend) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundFeeDividendGas, data)
}

func (md *MethodDexFundFeeDividend) GetReceiveQuota() uint64 {
	return dexFundFeeDividendReceiveGas
}

func (md *MethodDexFundFeeDividend) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	param := new(dex.ParamDexFundDividend)
	if err := ABIDexFund.UnpackMethod(param, MethodNameDexFundFeeDividend, block.Data); err != nil {
		return err
	}
	if currentPeriodId, err := dex.GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	} else if param.PeriodId >= currentPeriodId {
		return fmt.Errorf("dividend periodId not before current periodId")
	}
	if lastDividendId := dex.GetLastFeeDividendIdFromStorage(db); lastDividendId > 0 && param.PeriodId != lastDividendId+1 {
		return fmt.Errorf("dividend period id not equals to expected id %d", lastDividendId+1)
	}
	return nil
}

func (md MethodDexFundFeeDividend) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		err error
	)
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundFeeDividend, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if lastDividendId := dex.GetLastFeeDividendIdFromStorage(db); lastDividendId > 0 && param.PeriodId != lastDividendId+1 {
		return handleReceiveErr(db, fmt.Errorf("fee dividend period id not equals to expected id %d", lastDividendId+1))
	}
	if err = doDivideFees(db, param.PeriodId); err != nil {
		return handleReceiveErr(db, err)
	} else {
		dex.SaveLastDividendIdToStorage(db, param.PeriodId)
	}
	return []*SendBlock{}, nil
}

type MethodDexFundMinedVxDividend struct {
}

func (md *MethodDexFundMinedVxDividend) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundMinedVxDividend) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundMinedVxDividend) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundMinedVxDividendGas, data)
}

func (md *MethodDexFundMinedVxDividend) GetReceiveQuota() uint64 {
	return dexFundMinedVxDividendReceiveGas
}

func (md *MethodDexFundMinedVxDividend) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	param := new(dex.ParamDexFundDividend)
	if err := ABIDexFund.UnpackMethod(param, MethodNameDexFundMinedVxDividend, block.Data); err != nil {
		return err
	}
	if currentPeriodId, err := dex.GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	} else if param.PeriodId >= currentPeriodId {
		return fmt.Errorf("specified periodId for mined vx dividend not before current periodId")
	}
	if lastMinedVxDividendId := dex.GetLastMinedVxDividendIdFromStorage(db); lastMinedVxDividendId > 0 && param.PeriodId != lastMinedVxDividendId+1 {
		return fmt.Errorf("mined vx dividend period id not equals to expected id %d", lastMinedVxDividendId+1)
	}
	return nil
}

func (md MethodDexFundMinedVxDividend) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		err error
	)
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundMinedVxDividend, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if lastMinedVxDividendId := dex.GetLastMinedVxDividendIdFromStorage(db); lastMinedVxDividendId > 0 && param.PeriodId != lastMinedVxDividendId+1 {
		return handleReceiveErr(db, fmt.Errorf("mined vx dividend period id not equals to expected id %d", lastMinedVxDividendId+1))
	}
	vxTokenId := &types.TokenTypeId{}
	vxTokenId.SetBytes(dex.VxTokenBytes)
	vxBalance := db.GetBalance(&types.AddressDexFund, vxTokenId)
	if amtForFeePerMarket, amtForPledge, amtForViteLabs, success := dex.GetMindedVxAmt(vxBalance); !success {
		return handleReceiveErr(db, fmt.Errorf("no vx available for mine"))
	} else {
		if err = doDivideMinedVxForFee(db, param.PeriodId, amtForFeePerMarket); err != nil {
			return handleReceiveErr(db, err)
		}
		if err = doDivideMinedVxForPledge(db, amtForPledge); err != nil {
			return handleReceiveErr(db, err)
		}
		if err = doDivideMinedVxForViteLabs(db, amtForViteLabs); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	dex.SaveLastMinedVxDividendIdToStorage(db, param.PeriodId)
	return []*SendBlock{}, nil
}

type MethodDexFundNewMarket struct {
}

func (md *MethodDexFundNewMarket) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewMarket) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundNewMarket) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewMarketGas, data)
}

func (md *MethodDexFundNewMarket) GetReceiveQuota() uint64 {
	return dexFundNewMarketReceiveGas
}

func (md *MethodDexFundNewMarket) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundNewMarket)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewMarket, block.Data); err != nil {
		return err
	}
	if err = dex.CheckMarketParam(db, param, block.TokenId, block.Amount); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundNewMarket) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var err error
	param := new(dex.ParamDexFundNewMarket)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewMarket, sendBlock.Data); err != nil {
		return []*SendBlock{}, err
	}
	if mi, _ := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); mi != nil {
		return []*SendBlock{}, dex.TradeMarketExistsError
	}
	marketInfo := &dex.MarketInfo{}
	newMarketEvent := &dex.NewMarketEvent{}
	if err = dex.RenderMarketInfo(db, marketInfo, newMarketEvent, param, sendBlock.AccountAddress); err != nil {
		return []*SendBlock{}, err
	}
	exceedAmount := new(big.Int).Sub(sendBlock.Amount, dex.NewMarketFeeAmount)
	if exceedAmount.Sign() > 0 {
		if _, err = depositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, exceedAmount); err != nil {
			return []*SendBlock{}, err
		}
	}
	userFee := &dexproto.UserFeeSettle{}
	userFee.Address = sendBlock.AccountAddress.Bytes()
	userFee.Amount = dex.NewMarketFeeDividendAmount.Bytes()
	fee := &dexproto.FeeSettle{}
	fee.Token = ledger.ViteTokenId.Bytes()
	fee.UserFeeSettles = append(fee.UserFeeSettles, userFee)
	if err = settleFeeSum(db, []*dexproto.FeeSettle{fee}); err != nil {
		return []*SendBlock{}, err
	}
	if err = settleUserFees(db, fee); err != nil {
		return []*SendBlock{}, err
	}
	if err = dex.AddDonateFeeSum(db); err != nil {
		return []*SendBlock{}, err
	}
	if err = dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken); err != nil {
		return []*SendBlock{}, err
	}
	dex.AddNewMarketEventLog(db, newMarketEvent)
	return []*SendBlock{}, nil
}

func handleNewOrderFail(db vmctxt_interface.VmDatabase, orderInfo *dexproto.OrderInfo, errCode int) ([]*SendBlock, error) {
	orderInfo.Order.Status = dex.NewFailed
	dex.EmitOrderFailLog(db, orderInfo, errCode)
	return []*SendBlock{}, nil
}

func handleReceiveErr(db vmctxt_interface.VmDatabase, err error) ([]*SendBlock, error) {
	dex.EmitErrLog(db, err)
	return []*SendBlock{}, nil
}

func depositAccount(db vmctxt_interface.VmDatabase, address types.Address, tokenId types.TokenTypeId, amount *big.Int) (*dexproto.Account, error) {
	if dexFund, err := dex.GetUserFundFromStorage(db, address); err != nil {
		return nil, err
	} else {
		account, exists := dex.GetAccountByTokeIdFromFund(dexFund, tokenId)
		available := new(big.Int).SetBytes(account.Available)
		account.Available = available.Add(available, amount).Bytes()
		if !exists {
			dexFund.Accounts = append(dexFund.Accounts, account)
		}
		return account, dex.SaveUserFundToStorage(db, address, dexFund)
	}
}

func checkAndLockFundForNewOrder(dexFund *dex.UserFund, orderInfo *dexproto.OrderInfo) (needUpdate bool, err error) {
	var (
		lockToken, lockAmount []byte
		lockTokenId           *types.TokenTypeId
		lockAmountToInc       *big.Int
	)
	switch orderInfo.Order.Side {
	case false: //buy
		lockToken = orderInfo.OrderTokenInfo.QuoteToken
		if orderInfo.Order.Type == dex.Limited {
			lockAmount = dex.AddBigInt(orderInfo.Order.Amount, orderInfo.Order.LockedBuyFee)
		}
	case true: // sell
		lockToken = orderInfo.OrderTokenInfo.TradeToken
		lockAmount = orderInfo.Order.Quantity
	}
	if tkId, err := types.BytesToTokenTypeId(lockToken); err != nil {
		return false, err
	} else {
		lockTokenId = &tkId
	}
	//var tokenName string
	//if tokenInfo := cabi.GetTokenById(db, *lockTokenId); tokenInfo != nil {
	//	tokenName = tokenInfo.TokenName
	//}
	account, exists := dex.GetAccountByTokeIdFromFund(dexFund, *lockTokenId)
	available := new(big.Int).SetBytes(account.Available)
	if orderInfo.Order.Type != dex.Market || orderInfo.Order.Side { // limited or sell orderInfo
		lockAmountToInc = new(big.Int).SetBytes(lockAmount)
		//fmt.Printf("token %s, available %s , lockAmountToInc %s\n", tokenName, available.String(), lockAmountToInc.String())
		if available.Cmp(lockAmountToInc) < 0 {
			return false, fmt.Errorf("orderInfo lock amount exceed fund available")
		}
	}
	if !orderInfo.Order.Side && orderInfo.Order.Type == dex.Market { // buy or market orderInfo
		if available.Sign() <= 0 {
			return false, fmt.Errorf("no quote amount available for market sell orderInfo")
		} else {
			lockAmount = available.Bytes()
			//NOTE: use amount available for orderInfo amount to full fill
			orderInfo.Order.Amount = lockAmount
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

func doSettleFund(db vmctxt_interface.VmDatabase, action *dexproto.UserFundSettle) error {
	address := types.Address{}
	address.SetBytes([]byte(action.Address))
	if dexFund, err := dex.GetUserFundFromStorage(db, address); err != nil {
		return err
	} else {
		for _, fundSettle := range action.FundSettles {
			if tokenId, err := types.BytesToTokenTypeId(fundSettle.Token); err != nil {
				return err
			} else {
				if err, _ = dex.GetTokenInfo(db, tokenId); err != nil {
					return err
				}
				account, exists := dex.GetAccountByTokeIdFromFund(dexFund, tokenId)
				//fmt.Printf("origin account for :address %s, tokenId %s, available %s, locked %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(account.Available).String(), new(big.Int).SetBytes(account.Locked).String())
				if dex.CmpToBigZero(fundSettle.ReduceLocked) != 0 {
					if dex.CmpForBigInt(fundSettle.ReduceLocked, account.Locked) > 0 {
						return fmt.Errorf("try reduce locked amount execeed locked")
					}
					account.Locked = dex.SubBigIntAbs(account.Locked, fundSettle.ReduceLocked)
				}
				if dex.CmpToBigZero(fundSettle.ReleaseLocked) != 0 {
					if dex.CmpForBigInt(fundSettle.ReleaseLocked, account.Locked) > 0 {
						return fmt.Errorf("try release locked amount execeed locked")
					}
					account.Locked = dex.SubBigIntAbs(account.Locked, fundSettle.ReleaseLocked)
					account.Available = dex.AddBigInt(account.Available, fundSettle.ReleaseLocked)
				}
				if dex.CmpToBigZero(fundSettle.IncAvailable) != 0 {
					account.Available = dex.AddBigInt(account.Available, fundSettle.IncAvailable)
				}
				if !exists {
					dexFund.Accounts = append(dexFund.Accounts, account)
				}
				// must do after account updated by settle
				if bytes.Equal(fundSettle.Token, dex.VxTokenBytes) {
					if err = onSettleVx(db, action.Address, fundSettle, account); err != nil {
						return err
					}
				}
				//fmt.Printf("settle for :address %s, tokenId %s, ReduceLocked %s, ReleaseLocked %s, IncAvailable %s\n", address.String(), tokenId.String(), new(big.Int).SetBytes(action.ReduceLocked).String(), new(big.Int).SetBytes(action.ReleaseLocked).String(), new(big.Int).SetBytes(action.IncAvailable).String())
			}
		}
		if err = dex.SaveUserFundToStorage(db, address, dexFund); err != nil {
			return err
		}
	}
	return nil
}

func settleFeeSum(db vmctxt_interface.VmDatabase, feeActions []*dexproto.FeeSettle) error {
	var (
		feeSumByPeriod *dex.FeeSumByPeriod
		err            error
	)
	if feeSumByPeriod, err = dex.GetCurrentFeeSumFromStorage(db); err != nil {
		return err
	} else {
		if feeSumByPeriod == nil { // need roll period when current period feeSum not saved yet
			if formerPeriodId, err := rollFeeSumPeriodId(db); err != nil {
				return err
			} else {
				feeSumByPeriod = &dex.FeeSumByPeriod{}
				feeSumByPeriod.LastValidPeriod = formerPeriodId
			}
		}
		feeAmountMap := make(map[types.TokenTypeId][]byte)
		for _, feeAction := range feeActions {
			tokenId, _ := types.BytesToTokenTypeId(feeAction.Token)
			for _, feeAcc := range feeAction.UserFeeSettles {
				feeAmountMap[tokenId] = dex.AddBigInt(feeAmountMap[tokenId], feeAcc.Amount)
			}
		}

		for _, feeAcc := range feeSumByPeriod.Fees {
			tokenId, _ := types.BytesToTokenTypeId(feeAcc.Token)
			if _, ok := feeAmountMap[tokenId]; ok {
				feeAcc.Amount = dex.AddBigInt(feeAcc.Amount, feeAmountMap[tokenId])
				delete(feeAmountMap, tokenId)
			}
		}
		for tokenId, feeAmount := range feeAmountMap {
			newFeeAcc := &dexproto.FeeAccount{}
			newFeeAcc.Token = tokenId.Bytes()
			newFeeAcc.Amount = feeAmount
			feeSumByPeriod.Fees = append(feeSumByPeriod.Fees, newFeeAcc)
		}
	}
	if err = dex.SaveCurrentFeeSumToStorage(db, feeSumByPeriod); err != nil {
		return err
	} else {
		return nil
	}
}

func settleUserFees(db vmctxt_interface.VmDatabase, feeAction *dexproto.FeeSettle) error {
	var (
		userFees *dex.UserFees
		periodId uint64
		err      error
	)
	if periodId, err = dex.GetCurrentPeriodIdFromStorage(db); err != nil {
		return err
	}
	for _, userFeeSettle := range feeAction.UserFeeSettles {
		if userFees, err = dex.GetUserFeesFromStorage(db, userFeeSettle.Address); err != nil {
			return err
		}
		if userFees == nil || len(userFees.Fees) == 0 {
			userFees = &dex.UserFees{}
		}
		feeLen := len(userFees.Fees)
		if feeLen > 0 && periodId == userFees.Fees[feeLen-1].Period {
			var foundToken = false
			for _, feeAcc := range userFees.Fees[feeLen-1].UserFees {
				if bytes.Compare(feeAcc.Token, feeAction.Token) == 0 {
					feeAcc.Amount = dex.AddBigInt(feeAcc.Amount, userFeeSettle.Amount)
					foundToken = true
					break
				}
			}
			if !foundToken {
				feeAcc := &dexproto.FeeAccount{}
				feeAcc.Token = feeAction.Token
				feeAcc.Amount = userFeeSettle.Amount
				userFees.Fees[feeLen-1].UserFees = append(userFees.Fees[feeLen-1].UserFees, feeAcc)
			}
		} else {
			userFeeByPeriodId := &dexproto.UserFeeWithPeriod{}
			userFeeByPeriodId.Period = periodId
			feeAcc := &dexproto.FeeAccount{}
			feeAcc.Token = feeAction.Token
			feeAcc.Amount = userFeeSettle.Amount
			userFeeByPeriodId.UserFees = []*dexproto.FeeAccount{feeAcc}
			userFees.Fees = append(userFees.Fees, userFeeByPeriodId)
		}
		if err = dex.SaveUserFeesToStorage(db, userFeeSettle.Address, userFees); err != nil {
			return err
		}
	}
	return nil
}

func rollFeeSumPeriodId(db vmctxt_interface.VmDatabase) (uint64, error) {
	formerId := dex.GetFeeSumLastPeriodIdForRoll(db)
	if err := dex.SaveFeeSumLastPeriodIdForRoll(db); err != nil {
		return 0, err
	} else {
		return formerId, err
	}
}

func onDepositVx(db vmctxt_interface.VmDatabase, address types.Address, depositAmount *big.Int, updatedVxAccount *dexproto.Account) error {
	return doSettleVxFunds(db, address.Bytes(), depositAmount, updatedVxAccount)
}

func onWithdrawVx(db vmctxt_interface.VmDatabase, address types.Address, withdrawAmount *big.Int, updatedVxAccount *dexproto.Account) error {
	return doSettleVxFunds(db, address.Bytes(), new(big.Int).Neg(withdrawAmount), updatedVxAccount)
}

func onSettleVx(db vmctxt_interface.VmDatabase, address []byte, fundSettle *dexproto.FundSettle, updatedVxAccount *dexproto.Account) error {
	amtChange := dex.SubBigInt(fundSettle.IncAvailable, fundSettle.ReduceLocked)
	return doSettleVxFunds(db, address, amtChange, updatedVxAccount)
}

// only settle validAmount and amount changed from previous period
func doSettleVxFunds(db vmctxt_interface.VmDatabase, addressBytes []byte, amtChange *big.Int, updatedVxAccount *dexproto.Account) error {
	var (
		vxFunds               *dex.VxFunds
		userNewAmt, sumChange *big.Int
		err                   error
		periodId              uint64
		fundsLen              int
		needUpdate            bool
	)
	if vxFunds, err = dex.GetVxFundsFromStorage(db, addressBytes); err != nil {
		return err
	} else {
		if vxFunds == nil {
			vxFunds = &dex.VxFunds{}
		}
		if periodId, err = dex.GetCurrentPeriodIdFromStorage(db); err != nil {
			return err
		}
		fundsLen = len(vxFunds.Funds)
		userNewAmt = new(big.Int).SetBytes(dex.AddBigInt(updatedVxAccount.Available, updatedVxAccount.Locked))
		if fundsLen == 0 { //need append new period
			if dex.IsValidVxAmountForDividend(userNewAmt) {
				fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
				vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
				sumChange = userNewAmt
				needUpdate = true
			}
		} else if vxFunds.Funds[fundsLen-1].Period == periodId { //update current period
			if dex.IsValidVxAmountForDividend(userNewAmt) {
				if dex.IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
					sumChange = amtChange
				} else {
					sumChange = userNewAmt
				}
				vxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
			} else {
				if dex.IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
					sumChange = dex.NegativeAmount(vxFunds.Funds[fundsLen-1].Amount)
				}
				if fundsLen > 1 { // in case fundsLen > 1, update last period to diff the condition of current period not changed ever from last saved period
					vxFunds.Funds[fundsLen-1].Amount = userNewAmt.Bytes()
				} else { // clear funds in case only current period saved and not valid any more
					vxFunds.Funds = nil
				}
			}
			needUpdate = true
		} else { // need save new status, whether new amt is valid or not, in order to diff last saved period
			if dex.IsValidVxAmountForDividend(userNewAmt) {
				if dex.IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
					sumChange = amtChange
				} else {
					sumChange = userNewAmt
				}
				fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
				vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
				needUpdate = true
			} else {
				if dex.IsValidVxAmountBytesForDividend(vxFunds.Funds[fundsLen-1].Amount) {
					sumChange = dex.NegativeAmount(vxFunds.Funds[fundsLen-1].Amount)
					fundWithPeriod := &dexproto.VxFundWithPeriod{Period: periodId, Amount: userNewAmt.Bytes()}
					vxFunds.Funds = append(vxFunds.Funds, fundWithPeriod)
					needUpdate = true
				}
			}
		}
	}
	if len(vxFunds.Funds) > 0 && needUpdate {
		if err = dex.SaveVxFundsToStorage(db, addressBytes, vxFunds); err != nil {
			return err
		}
	} else if len(vxFunds.Funds) == 0 && fundsLen > 0 {
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
			sumFundsLen := len(vxSumFunds.Funds)
			if sumFundsLen == 0 {
				if sumChange.Sign() > 0 {
					vxSumFunds.Funds = append(vxSumFunds.Funds, &dexproto.VxFundWithPeriod{Period: periodId, Amount: sumChange.Bytes()})
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
		feeSumsMap    map[uint64]*dex.FeeSumByPeriod
		donateFeeSums = make(map[uint64]*big.Int)
		vxSumFunds    *dex.VxFunds
		err           error
	)

	//allow divide history fees that not divided yet
	if feeSumsMap, donateFeeSums, err = dex.GetNotDividedFeeSumsByPeriodIdFromStorage(db, periodId); err != nil {
		return err
	} else if len(feeSumsMap) == 0 || len(feeSumsMap) > 4 { // no fee to divide, or fee types more than 4
		return nil
	}

	if vxSumFunds, err = dex.GetVxSumFundsFromStorage(db); err != nil {
		return err
	} else if vxSumFunds == nil {
		return nil
	}
	foundVxSumFunds, vxSumAmtBytes, needUpdateVxSum, _ := dex.MatchVxFundsByPeriod(vxSumFunds, periodId, false)
	//fmt.Printf("foundVxSumFunds %v, vxSumAmtBytes %s, needUpdateVxSum %v with periodId %d\n", foundVxSumFunds, new(big.Int).SetBytes(vxSumAmtBytes).String(), needUpdateVxSum, periodId)
	if !foundVxSumFunds { // not found vxSumFunds
		return nil
	}
	if needUpdateVxSum {
		if err := dex.SaveVxSumFundsToStorage(db, vxSumFunds); err != nil {
			return err
		}
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
				if amt, ok := feeSumMap[tokenId]; !ok {
					feeSumMap[tokenId] = new(big.Int).SetBytes(feeAccount.Amount)
				} else {
					feeSumMap[tokenId] = amt.Add(amt, new(big.Int).SetBytes(feeAccount.Amount))
				}
			}
		}

		dex.MarkFeeSumAsFeeDivided(db, fee, pId)
	}

	for pIdForDonateFee, donateFeeSum := range donateFeeSums {
		feeSumMap[ledger.ViteTokenId] = new(big.Int).Add(feeSumMap[ledger.ViteTokenId], donateFeeSum)
		dex.DeleteDonateFeeSum(db, pIdForDonateFee)
	}

	var (
		userVxFundsKey, userVxFundsBytes []byte
		ok                               bool
	)

	iterator := db.NewStorageIterator(&types.AddressDexFund, dex.VxFundKeyPrefix)

	feeSumLeavedMap := make(map[types.TokenTypeId]*big.Int)
	dividedVxAmtMap := make(map[types.TokenTypeId]*big.Int)
	for {
		if len(feeSumMap) == 0 {
			break
		}
		if userVxFundsKey, userVxFundsBytes, ok = iterator.Next(); !ok {
			break
		}

		addressBytes := userVxFundsKey[len(dex.VxFundKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		userVxFunds := &dex.VxFunds{}
		if userVxFunds, err = userVxFunds.DeSerialize(userVxFundsBytes); err != nil {
			return err
		}

		var userFeeDividend = make(map[types.TokenTypeId]*big.Int)
		foundVxFunds, userVxAmtBytes, needUpdateVxFunds, needDeleteVxFunds := dex.MatchVxFundsByPeriod(userVxFunds, periodId, true)
		if !foundVxFunds {
			continue
		}
		if needDeleteVxFunds {
			dex.DeleteVxFundsFromStorage(db, address.Bytes())
		} else if needUpdateVxFunds {
			if err = dex.SaveVxFundsToStorage(db, address.Bytes(), userVxFunds); err != nil {
				return err
			}
		}
		userVxAmount := new(big.Int).SetBytes(userVxAmtBytes)
		//fmt.Printf("address %s, userVxAmount %s, needDeleteVxFunds %v\n", string(address.Bytes()), userVxAmount.String(), needDeleteVxFunds)
		if !dex.IsValidVxAmountForDividend(userVxAmount) { //skip vxAmount not valid for dividend
			continue
		}

		var finished bool
		for tokenId, feeSumAmount := range feeSumMap {
			if _, ok = feeSumLeavedMap[tokenId]; !ok {
				feeSumLeavedMap[tokenId] = new(big.Int).Set(feeSumAmount)
				dividedVxAmtMap[tokenId] = big.NewInt(0)
			}
			//fmt.Printf("tokenId %s, address %s, vxSumAmt %s, userVxAmount %s, dividedVxAmt %s, toDivideFeeAmt %s, toDivideLeaveAmt %s\n", tokenId.String(), address.String(), vxSumAmt.String(), userVxAmount.String(), dividedVxAmtMap[tokenId], toDivideFeeAmt.String(), toDivideLeaveAmt.String())
			userFeeDividend[tokenId], finished = dex.DivideByProportion(vxSumAmt, userVxAmount, dividedVxAmtMap[tokenId], feeSumAmount, feeSumLeavedMap[tokenId])
			if finished {
				delete(feeSumMap, tokenId)
			}
		}
		if err = dex.BatchSaveUserFund(db, address, userFeeDividend); err != nil {
			return err
		}
	}
	return nil
}

func doDivideMinedVxForFee(db vmctxt_interface.VmDatabase, periodId uint64, minedVxAmtPerMarket *big.Int) error {
	var (
		feeSum                *dex.FeeSumByPeriod
		feeSumMap             = make(map[types.TokenTypeId]*big.Int)
		dividedFeeMap         = make(map[types.TokenTypeId]*big.Int)
		toDivideVxLeaveAmtMap = make(map[types.TokenTypeId]*big.Int)
		tokenId               types.TokenTypeId
		err                   error
	)
	if feeSum, err = dex.GetFeeSumByPeriodIdFromStorage(db, periodId); err != nil {
		return err
	} else if feeSum == nil { // no fee to divide
		return nil
	}
	for _, feeSum := range feeSum.Fees {
		if tokenId, err = types.BytesToTokenTypeId(feeSum.Token); err != nil {
			return err
		}
		feeSumMap[tokenId] = new(big.Int).SetBytes(feeSum.Amount)
		toDivideVxLeaveAmtMap[tokenId] = minedVxAmtPerMarket
		dividedFeeMap[tokenId] = big.NewInt(0)
	}

	dex.MarkFeeSumAsMinedVxDivided(db, feeSum, periodId)

	var (
		userFeesKey, userFeesBytes []byte
		ok                         bool
	)

	vxTokenId := types.TokenTypeId{}
	vxTokenId.SetBytes(dex.VxTokenBytes)
	iterator := db.NewStorageIterator(&types.AddressDexFund, dex.UserFeeKeyPrefix)
	for {
		if userFeesKey, userFeesBytes, ok = iterator.Next(); !ok {
			break
		}

		addressBytes := userFeesKey[len(dex.UserFeeKeyPrefix):]
		address := types.Address{}
		if err = address.SetBytes(addressBytes); err != nil {
			return err
		}
		userFees := &dex.UserFees{}
		if userFees, err = userFees.DeSerialize(userFeesBytes); err != nil {
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
					vxDividend, finished := dex.DivideByProportion(feeSumAmt, new(big.Int).SetBytes(userFee.Amount), dividedFeeMap[tokenId], minedVxAmtPerMarket, toDivideVxLeaveAmtMap[tokenId])
					userVxDividend.Add(userVxDividend, vxDividend)
					if finished {
						delete(feeSumMap, tokenId)
					}
				}
			}
			if err = dex.BatchSaveUserFund(db, address, map[types.TokenTypeId]*big.Int{vxTokenId: userVxDividend}); err != nil {
				return err
			}
		}
		if len(userFees.Fees) == 1 {
			dex.DeleteUserFeesFromStorage(db, addressBytes)
		} else {
			userFees.Fees = userFees.Fees[1:]
			if err = dex.SaveUserFeesToStorage(db, addressBytes, userFees); err != nil {
				return err
			}
		}
	}
	return nil
}

func doDivideMinedVxForPledge(db vmctxt_interface.VmDatabase, minedVxAmt *big.Int) error {
	// support accumulate history pledge vx
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	return nil
}

func doDivideMinedVxForViteLabs(db vmctxt_interface.VmDatabase, minedVxAmt *big.Int) error {
	if minedVxAmt.Sign() == 0 {
		return nil
	}
	return nil
}
