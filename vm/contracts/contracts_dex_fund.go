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
		{"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"orderId","type":"bytes"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}, {"name":"orderType", "type":"int8"}, {"name":"price", "type":"string"}, {"name":"quantity", "type":"uint256"}]},
		{"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexFundFeeDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundMinedVxDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundNewMarket", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
		{"type":"function","name":"DexFundSetOwner", "inputs":[{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"DexFundConfigMineMarket", "inputs":[{"name":"allowMine","type":"bool"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
		{"type":"function","name":"DexFundPledgeForVx", "inputs":[{"name":"actionType","type":"int8"}, {"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundPledgeForVip", "inputs":[{"name":"actionType","type":"int8"}]},
		{"type":"function","name":"PledgeCallback", "inputs":[{"name":"success","type":"bool"}]},
		{"type":"function","name":"CancelPledgeCallback", "inputs":[{"name":"success","type":"bool"}]}
	]`

	MethodNameDexFundUserDeposit          = "DexFundUserDeposit"
	MethodNameDexFundUserWithdraw         = "DexFundUserWithdraw"
	MethodNameDexFundNewOrder             = "DexFundNewOrder"
	MethodNameDexFundSettleOrders         = "DexFundSettleOrders"
	MethodNameDexFundFeeDividend          = "DexFundFeeDividend"
	MethodNameDexFundMinedVxDividend      = "DexFundMinedVxDividend"
	MethodNameDexFundNewMarket            = "DexFundNewMarket"
	MethodNameDexFundSetOwner             = "DexFundSetOwner"
	MethodNameDexFundConfigMineMarket     = "DexFundConfigMineMarket"
	MethodNameDexFundPledgeForVx          = "DexFundPledgeForVx"
	MethodNameDexFundPledgeForVip         = "DexFundPledgeForVip"
	MethodNameDexFundPledgeCallback       = "PledgeCallback"
	MethodNameDexFundCancelPledgeCallback = "CancelPledgeCallback"
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
	return nil
}

func (md *MethodDexFundUserDeposit) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	if account, err := dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount); err != nil {
		return []*SendBlock{}, err
	} else {
		// must do after account updated by deposit
		if bytes.Equal(sendBlock.TokenId.Bytes(), dex.VxTokenBytes) {
			if err = dex.OnDepositVx(db, sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
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
	if tokenInfo, _ := dex.GetTokenInfo(db, param.Token); tokenInfo == nil {
		return handleReceiveErr(db, dex.InvalidTokenErr)
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
		if err = dex.OnWithdrawVx(db, sendBlock.AccountAddress, param.Amount, account); err != nil {
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
	if err = dex.PreCheckOrderParam(param); err != nil {
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
	if _, err = dex.CheckAndLockFundForNewOrder(dexFund, orderInfo); err != nil {
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
		if err = dex.DoSettleFund(db, fundAction); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	if len(settleActions.FeeActions) > 0 {
		if err = dex.SettleFeeSum(db, settleActions.FeeActions); err != nil {
			return handleReceiveErr(db, err)
		}
		for _, feeAction := range settleActions.FeeActions {
			if err = dex.SettleUserFees(db, feeAction); err != nil {
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
	if err = dex.DoDivideFees(db, param.PeriodId); err != nil {
		return handleReceiveErr(db, err)
	} else {
		dex.SaveLastFeeDividendIdToStorage(db, param.PeriodId)
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
		if err = dex.DoDivideMinedVxForFee(db, param.PeriodId, amtForFeePerMarket); err != nil {
			return handleReceiveErr(db, err)
		}
		if err = dex.DoDivideMinedVxForPledge(db, amtForPledge); err != nil {
			return handleReceiveErr(db, err)
		}
		if err = dex.DoDivideMinedVxForViteLabs(db, amtForViteLabs); err != nil {
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
	if err = dex.CheckMarketParam(param, block.TokenId, block.Amount); err != nil {
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
		if _, err = dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, exceedAmount); err != nil {
			return []*SendBlock{}, err
		}
	}
	userFee := &dexproto.UserFeeSettle{}
	userFee.Address = sendBlock.AccountAddress.Bytes()
	userFee.Amount = dex.NewMarketFeeDividendAmount.Bytes()
	fee := &dexproto.FeeSettle{}
	fee.Token = ledger.ViteTokenId.Bytes()
	fee.UserFeeSettles = append(fee.UserFeeSettles, userFee)
	if err = dex.SettleFeeSum(db, []*dexproto.FeeSettle{fee}); err != nil {
		return []*SendBlock{}, err
	}
	if err = dex.SettleUserFees(db, fee); err != nil {
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

type MethodDexFundSetOwner struct {
}

func (md *MethodDexFundSetOwner) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSetOwner) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundSetOwner) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundSetOwnerGas, data)
}

func (md *MethodDexFundSetOwner) GetReceiveQuota() uint64 {
	return dexFundSetOwnerReceiveGas
}

func (md *MethodDexFundSetOwner) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	if err = ABIDexFund.UnpackMethod(new(dex.ParamDexFundSetOwner), MethodNameDexFundSetOwner, block.Data); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundSetOwner) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var err error
	var param = new(dex.ParamDexFundSetOwner)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSetOwner, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		dex.SetOwner(db, param.NewOwner)
	} else {
		handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	return []*SendBlock{}, nil
}

type MethodDexFundConfigMineMarket struct {
}

func (md *MethodDexFundConfigMineMarket) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundConfigMineMarket) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundConfigMineMarket) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundConfigMineMarketGas, data)
}

func (md *MethodDexFundConfigMineMarket) GetReceiveQuota() uint64 {
	return dexFundConfigMineMarketReceiveGas
}

func (md *MethodDexFundConfigMineMarket) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var err error
	if err = ABIDexFund.UnpackMethod(new(dex.ParamDexFundConfigMineMarket), MethodNameDexFundConfigMineMarket, block.Data); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundConfigMineMarket) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var err error
	var param = new(dex.ParamDexFundConfigMineMarket)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundConfigMineMarket, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if marketInfo, err := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); marketInfo != nil && err == nil {
			if param.AllowMine != marketInfo.AllowMine {
				marketInfo.AllowMine = param.AllowMine
				if err = dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken); err != nil {
					return handleReceiveErr(db, err)
				}
			} else {
				if marketInfo.AllowMine {
					return handleReceiveErr(db, dex.TradeMarketAllowMineError)
				} else {
					return handleReceiveErr(db, dex.TradeMarketNotAllowMineError)
				}
			}
		} else {
			return handleReceiveErr(db, dex.TradeMarketNotExistsError)
		}
	} else {
		return handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	return []*SendBlock{}, nil
}

type MethodDexFundPledgeForVx struct {
}

func (md *MethodDexFundPledgeForVx) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForVx) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundPledgeForVx) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundPledgeForVxGas, data)
}

func (md *MethodDexFundPledgeForVx) GetReceiveQuota() uint64 {
	return dexFundPledgeForVxReceiveGas
}

func (md *MethodDexFundPledgeForVx) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamDexFundPledgeForVx)
	)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundPledgeForVx, block.Data); err != nil {
		return err
	} else {
		if param.Amount.Cmp(dex.PledgeForVxMinAmount) < 0 {
			return dex.InvalidPledgeAmountErr
		}
		if param.ActionType != dex.Pledge && param.ActionType != dex.CancelPledge {
			return dex.InvalidPledgeActionTypeErr
		}
	}
	return nil
}

func (md MethodDexFundPledgeForVx) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var param = new(dex.ParamDexFundPledgeForVx)
	if err := ABIDexFund.UnpackMethod(param, MethodNameDexFundPledgeForVx, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	return handlePledgeAction(db, block, dex.PledgeForVx, param.ActionType, sendBlock.AccountAddress, param.Amount)
}

type MethodDexFundPledgeForVip struct {
}

func (md *MethodDexFundPledgeForVip) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForVip) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundPledgeForVip) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundPledgeForVipGas, data)
}

func (md *MethodDexFundPledgeForVip) GetReceiveQuota() uint64 {
	return dexFundPledgeForVipReceiveGas
}

func (md *MethodDexFundPledgeForVip) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamDexFundPledgeForVip)
	)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundPledgeForVip, block.Data); err != nil {
		return err
	}
	if param.ActionType != dex.Pledge && param.ActionType != dex.CancelPledge {
		return dex.InvalidPledgeActionTypeErr
	}
	return nil
}

func (md MethodDexFundPledgeForVip) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var param = new(dex.ParamDexFundPledgeForVip)
	if err := ABIDexFund.UnpackMethod(param, MethodNameDexFundPledgeForVip, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	return handlePledgeAction(db, block, dex.PledgeForVip, param.ActionType, sendBlock.AccountAddress, dex.PledgeForVipAmount)
}

type MethodDexFundPledgeCallback struct {
}

func (md *MethodDexFundPledgeCallback) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeCallback) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundPledgeCallback) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundPledgeCallbackGas, data)
}

func (md *MethodDexFundPledgeCallback) GetReceiveQuota() uint64 {
	return dexFundPledgeCallbackReceiveGas
}

func (md *MethodDexFundPledgeCallback) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidPledgeSourceAddressErr
	}
	if err := ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), MethodNameDexFundPledgeCallback, block.Data); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundPledgeCallback) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		err           error
		callbackParam = new(dex.ParamDexFundPledgeCallBack)
	)
	if err = ABIDexFund.UnpackMethod(callbackParam, MethodNameDexFundPledgeCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	callSendBlock := &ledger.AccountBlock{}
	pledgeParam := new(dex.ParamDexFundPledge)
	if err = cabi.ABIPledge.UnpackMethod(pledgeParam, cabi.MethodNamePledge, callSendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if callbackParam.Success {
		if pledgeParam.PledgeType == dex.PledgeForVip {
			var pledgeVip *dex.PledgeVip
			if pledgeVip, _ = dex.GetPledgeForVip(db, pledgeParam.Source); pledgeVip != nil {
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes + 1
				if err = dex.SavePledgeForVip(db, pledgeParam.Source, pledgeVip); err != nil {
					return handleReceiveErr(db, err)
				}
				return doCancelPledge(db, block, pledgeParam.Source, pledgeParam.PledgeType, pledgeParam.Amount)
			} else {
				pledgeVip = &dex.PledgeVip{}
				pledgeVip.Timestamp = dex.GetTimestampInt64(db)
				pledgeVip.PledgeTimes = 1
				if err = dex.SavePledgeForVip(db, pledgeParam.Source, pledgeVip); err != nil {
					return handleReceiveErr(db, err)
				}
			}
		} else {
			pledgeAmount := dex.GetPledgeForVx(db, pledgeParam.Source)
			pledgeAmount.Add(pledgeAmount, pledgeParam.Amount)
			dex.SavePledgeForVx(db, pledgeParam.Source, pledgeAmount)
		}
	} else {
		if _, err = dex.DepositAccount(db, pledgeParam.Source, ledger.ViteTokenId, pledgeParam.Amount); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	return []*SendBlock{}, nil
}

type MethodDexFundCancelPledgeCallback struct {
}

func (md *MethodDexFundCancelPledgeCallback) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelPledgeCallback) GetRefundData() []byte {
	return []byte{}
}

func (md *MethodDexFundCancelPledgeCallback) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundCancelPledgeCallbackGas, data)
}

func (md *MethodDexFundCancelPledgeCallback) GetReceiveQuota() uint64 {
	return dexFundCancelPledgeCallbackReceiveGas
}

func (md *MethodDexFundCancelPledgeCallback) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidPledgeSourceAddressErr
	}
	if err := ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), MethodNameDexFundCancelPledgeCallback, block.Data); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundCancelPledgeCallback) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	var (
		err           error
		callbackParam = new(dex.ParamDexFundPledgeCallBack)
	)
	if err = ABIDexFund.UnpackMethod(callbackParam, MethodNameDexFundCancelPledgeCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	callSendBlock := &ledger.AccountBlock{}
	cancelPledgeParam := new(dex.ParamDexFundPledge)
	if err = cabi.ABIPledge.UnpackMethod(cancelPledgeParam, cabi.MethodNameCancelPledge, callSendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if callbackParam.Success {
		if sendBlock.Amount.Cmp(cancelPledgeParam.Amount) != 0 {
			return handleReceiveErr(db, dex.InvalidAmountForCancelPledgeErr)
		}
		if cancelPledgeParam.PledgeType == dex.PledgeForVip {
			var pledgeVip *dex.PledgeVip
			if pledgeVip, _ = dex.GetPledgeForVip(db, cancelPledgeParam.Source); pledgeVip != nil {
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes - 1
				if pledgeVip.PledgeTimes == 0 {
					dex.DeletePledgeForVip(db, cancelPledgeParam.Source)
				} else {
					if err = dex.SavePledgeForVip(db, cancelPledgeParam.Source, pledgeVip); err != nil {
						return handleReceiveErr(db, err)
					}
				}
			} else {
				return handleReceiveErr(db, dex.PledgeForVipNotExistsErr)
			}
		} else {
			pledgeAmount := dex.GetPledgeForVx(db, cancelPledgeParam.Source)
			leave := new(big.Int).Sub(pledgeAmount, cancelPledgeParam.Amount)
			if leave.Sign() < 0 {
				return handleReceiveErr(db, dex.InvalidAmountForCancelPledgeErr)
			} else if leave.Sign() == 0 {
				dex.DeletePledgeForVx(db, cancelPledgeParam.Source)
			} else {
				dex.SavePledgeForVx(db, cancelPledgeParam.Source, leave)
			}
		}
		if _, err = dex.DepositAccount(db, cancelPledgeParam.Source, ledger.ViteTokenId, cancelPledgeParam.Amount); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	return []*SendBlock{}, nil
}

func handlePledgeAction(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, pledgeType int8, actionType int8, address types.Address, amount *big.Int) ([]*SendBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if actionType == dex.Pledge {
		if methodData, err = dex.PledgeRequest(db, address, pledgeType, amount); err != nil {
			return handleReceiveErr(db, err)
		} else {
			return []*SendBlock{
				{
					block,
					types.AddressPledge,
					ledger.BlockTypeSendCall,
					amount,
					ledger.ViteTokenId, // no need send token
					methodData,
				},
			}, nil
		}
	} else {
		return doCancelPledge(db, block, address, pledgeType, amount)
	}
}

func doCancelPledge(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, address types.Address, pledgeType int8, amount *big.Int) ([]*SendBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if methodData, err = dex.CancelPledgeRequest(db, address, pledgeType, amount); err != nil {
		return handleReceiveErr(db, err)
	} else {
		return []*SendBlock{
			{
				block,
				types.AddressPledge,
				ledger.BlockTypeSendCall,
				big.NewInt(0),
				ledger.ViteTokenId, // no need send token
				methodData,
			},
		}, nil
	}
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
