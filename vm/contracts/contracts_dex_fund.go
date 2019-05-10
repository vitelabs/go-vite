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
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"strings"
)

const (
	jsonDexFund = `
	[
		{"type":"function","name":"DexFundUserDeposit", "inputs":[]},
		{"type":"function","name":"DexFundUserWithdraw", "inputs":[{"name":"token","type":"tokenId"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundNewOrder", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}, {"name":"side", "type":"bool"}, {"name":"orderType", "type":"int8"}, {"name":"price", "type":"string"}, {"name":"quantity", "type":"uint256"}]},
		{"type":"function","name":"DexFundSettleOrders", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexFundFeeDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundMinedVxDividend", "inputs":[{"name":"periodId","type":"uint64"}]},
		{"type":"function","name":"DexFundNewMarket", "inputs":[{"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
		{"type":"function","name":"DexFundSetOwner", "inputs":[{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"DexFundConfigMineMarket", "inputs":[{"name":"allowMine","type":"bool"}, {"name":"tradeToken","type":"tokenId"}, {"name":"quoteToken","type":"tokenId"}]},
		{"type":"function","name":"DexFundPledgeForVx", "inputs":[{"name":"actionType","type":"int8"}, {"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundPledgeForVip", "inputs":[{"name":"actionType","type":"int8"}]},
		{"type":"function","name":"AgentPledgeCallback", "inputs":[{"name":"success","type":"bool"}]},
		{"type":"function","name":"AgentCancelPledgeCallback", "inputs":[{"name":"success","type":"bool"}]},
		{"type":"function","name":"GetTokenInfoCallback", "inputs":[{"name":"exist","type":"bool"},{"name":"decimals","type":"uint8"},{"name":"tokenSymbol","type":"string"},{"name":"index","type":"uint16"}]},
		{"type":"function","name":"DexFundConfigTimerAddress", "inputs":[{"name":"address","type":"address"}]},
		{"type":"function","name":"NotifyTime", "inputs":[{"name":"timestamp","type":"int64"}]}
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
	MethodNameDexFundPledgeCallback       = "AgentPledgeCallback"
	MethodNameDexFundCancelPledgeCallback = "AgentCancelPledgeCallback"
	MethodNameDexFundGetTokenInfoCallback = "GetTokenInfoCallback"
	MethodNameDexFundConfigTimerAddress   = "DexFundConfigTimerAddress"
	MethodNameDexFundNotifyTime           = "NotifyTime"
)

var (
	ABIDexFund, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
)

type MethodDexFundUserDeposit struct {
}

func (md *MethodDexFundUserDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserDeposit) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundUserDeposit) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundDepositGas, data)
}

func (md *MethodDexFundUserDeposit) GetReceiveQuota() uint64 {
	return dexFundDepositReceiveGas
}

func (md *MethodDexFundUserDeposit) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 {
		return fmt.Errorf("deposit amount is zero")
	}
	return nil
}

func (md *MethodDexFundUserDeposit) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	if account, err := dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount); err != nil {
		return nil, err
	} else {
		// must do after account updated by deposit
		if bytes.Equal(sendBlock.TokenId.Bytes(), dex.VxTokenBytes) {
			if err = dex.OnDepositVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserWithdraw) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundUserWithdraw) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundWithdrawGas, data)
}

func (md *MethodDexFundUserWithdraw) GetReceiveQuota() uint64 {
	return dexFundWithdrawReceiveGas
}

func (md *MethodDexFundUserWithdraw) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md *MethodDexFundUserWithdraw) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
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
		if err = dex.OnWithdrawVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, param.Amount, account); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	if err = dex.SaveUserFundToStorage(db, sendBlock.AccountAddress, dexFund); err != nil {
		return handleReceiveErr(db, err)
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         param.Amount,
			TokenId:        param.Token,
			Data:           []byte{},
		},
	}, nil
}

type MethodDexFundNewOrder struct {
}

func (md *MethodDexFundNewOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewOrder) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewOrderGas, data)
}

func (md *MethodDexFundNewOrder) GetReceiveQuota() uint64 {
	return dexFundNewOrderReceiveGas
}

func (md *MethodDexFundNewOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md *MethodDexFundNewOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
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
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      types.AddressDexTrade,
			BlockType:      ledger.BlockTypeSendCall,
			TokenId:        ledger.ViteTokenId,
			Amount:         big.NewInt(0),
			Data:           tradeBlockData,
		},
	}, nil
}

type MethodDexFundSettleOrders struct {
}

func (md *MethodDexFundSettleOrders) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleOrders) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundSettleOrders) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundSettleOrdersGas, data)
}

func (md *MethodDexFundSettleOrders) GetReceiveQuota() uint64 {
	return dexFundSettleOrdersReceiveGas
}

func (md *MethodDexFundSettleOrders) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md MethodDexFundSettleOrders) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
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
		if err = dex.DoSettleFund(db, vm.ConsensusReader(), fundAction); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	if len(settleActions.FeeActions) > 0 {
		if err = dex.SettleFeeSum(db, vm.ConsensusReader(), settleActions.FeeActions); err != nil {
			return handleReceiveErr(db, err)
		}
		for _, feeAction := range settleActions.FeeActions {
			if err = dex.SettleUserFees(db, vm.ConsensusReader(), feeAction); err != nil {
				return handleReceiveErr(db, err)
			}
		}
	}
	return nil, nil
}

type MethodDexFundFeeDividend struct {
}

func (md *MethodDexFundFeeDividend) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundFeeDividend) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundFeeDividend) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundFeeDividendGas, data)
}

func (md *MethodDexFundFeeDividend) GetReceiveQuota() uint64 {
	return dexFundFeeDividendReceiveGas
}

func (md *MethodDexFundFeeDividend) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundDividend), MethodNameDexFundFeeDividend, block.Data)
}

func (md MethodDexFundFeeDividend) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err error
	)
	if !dex.IsOwner(db, sendBlock.AccountAddress) {
		return handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundFeeDividend, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if param.PeriodId >= dex.GetCurrentPeriodIdFromStorage(db, vm.ConsensusReader()) {
		return handleReceiveErr(db, fmt.Errorf("dividend periodId not before current periodId"))
	}
	if lastDividendId := dex.GetLastFeeDividendIdFromStorage(db); lastDividendId > 0 && param.PeriodId != lastDividendId+1 {
		return handleReceiveErr(db, fmt.Errorf("fee dividend period id not equals to expected id %d", lastDividendId+1))
	}
	if err = dex.DoDivideFees(db, param.PeriodId); err != nil {
		return handleReceiveErr(db, err)
	} else {
		if err = dex.SaveLastFeeDividendIdToStorage(db, param.PeriodId); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	return nil, nil
}

type MethodDexFundMinedVxDividend struct {
}

func (md *MethodDexFundMinedVxDividend) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundMinedVxDividend) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundMinedVxDividend) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundMinedVxDividendGas, data)
}

func (md *MethodDexFundMinedVxDividend) GetReceiveQuota() uint64 {
	return dexFundMinedVxDividendReceiveGas
}

func (md *MethodDexFundMinedVxDividend) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundDividend), MethodNameDexFundMinedVxDividend, block.Data)
}

func (md MethodDexFundMinedVxDividend) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		vxBalance *big.Int
		err       error
	)
	if !dex.IsOwner(db, sendBlock.AccountAddress) {
		return handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	param := new(dex.ParamDexFundDividend)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundMinedVxDividend, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if param.PeriodId >= dex.GetCurrentPeriodIdFromStorage(db, vm.ConsensusReader()) {
		return handleReceiveErr(db, fmt.Errorf("specified periodId for mined vx dividend not before current periodId"))
	}
	if lastMinedVxDividendId := dex.GetLastMinedVxDividendIdFromStorage(db); lastMinedVxDividendId > 0 && param.PeriodId != lastMinedVxDividendId+1 {
		return handleReceiveErr(db, fmt.Errorf("mined vx dividend period id not equals to expected id %d", lastMinedVxDividendId+1))
	}
	vxTokenId := &types.TokenTypeId{}
	vxTokenId.SetBytes(dex.VxTokenBytes)
	if vxBalance, err = db.GetBalance(vxTokenId); err != nil {
		return handleReceiveErr(db, err)
	}
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
	if err = dex.SaveLastMinedVxDividendIdToStorage(db, param.PeriodId); err != nil {
		return handleReceiveErr(db, err)
	}
	return nil, nil
}

type MethodDexFundNewMarket struct {
}

func (md *MethodDexFundNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewMarket) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewMarket) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewMarketGas, data)
}

func (md *MethodDexFundNewMarket) GetReceiveQuota() uint64 {
	return dexFundNewMarketReceiveGas
}

func (md *MethodDexFundNewMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md MethodDexFundNewMarket) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	param := new(dex.ParamDexFundNewMarket)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundNewMarket, sendBlock.Data); err != nil {
		return nil, err
	}
	if mi, _ := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); mi != nil {
		return nil, dex.TradeMarketExistsError
	}
	marketInfo := &dex.MarketInfo{}
	newMarketEvent := &dex.NewMarketEvent{}
	if err = dex.RenderMarketInfo(db, marketInfo, newMarketEvent, param.TradeToken, param.QuoteToken, nil, &sendBlock.AccountAddress); err != nil {
		return nil, err
	}
	exceedAmount := new(big.Int).Sub(sendBlock.Amount, dex.NewMarketFeeAmount)
	if exceedAmount.Sign() > 0 {
		if _, err = dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, exceedAmount); err != nil {
			return nil, err
		}
	}
	if marketInfo.Valid {
		if err = dex.OnNewMarketValid(db, vm.ConsensusReader(), marketInfo, newMarketEvent, param.TradeToken, param.QuoteToken, &sendBlock.AccountAddress); err != nil {
			return nil, err
		}
	} else {
		if getTokenInfoData, err := dex.OnNewMarketPending(db, param, marketInfo); err != nil {
			return nil, err
		} else {
			return []*ledger.AccountBlock{
				{
					AccountAddress: types.AddressDexFund,
					ToAddress:      types.AddressMintage,
					BlockType:      ledger.BlockTypeSendCall,
					//TODO check TokenId and Amount force set
					TokenId: ledger.ViteTokenId,
					Amount:  big.NewInt(0),
					Data:    getTokenInfoData,
				},
			}, nil
		}
	}
	return nil, nil
}

type MethodDexFundSetOwner struct {
}

func (md *MethodDexFundSetOwner) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSetOwner) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundSetOwner) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundSetOwnerGas, data)
}

func (md *MethodDexFundSetOwner) GetReceiveQuota() uint64 {
	return dexFundSetOwnerReceiveGas
}

func (md *MethodDexFundSetOwner) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundSetOwner), MethodNameDexFundSetOwner, block.Data)
}

func (md MethodDexFundSetOwner) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexFundSetOwner)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundSetOwner, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if err = dex.SetOwner(db, param.NewOwner); err != nil {
			return handleReceiveErr(db, err)
		}
	} else {
		return handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	return nil, nil
}

type MethodDexFundConfigMineMarket struct {
}

func (md *MethodDexFundConfigMineMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundConfigMineMarket) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundConfigMineMarket) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundConfigMineMarketGas, data)
}

func (md *MethodDexFundConfigMineMarket) GetReceiveQuota() uint64 {
	return dexFundConfigMineMarketReceiveGas
}

func (md *MethodDexFundConfigMineMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundConfigMineMarket), MethodNameDexFundConfigMineMarket, block.Data)
}

func (md MethodDexFundConfigMineMarket) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexFundConfigMineMarket)
	if err = ABIDexFund.UnpackMethod(param, MethodNameDexFundConfigMineMarket, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if marketInfo, err := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); err == nil {
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
			return handleReceiveErr(db, err)
		}
	} else {
		return handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	return nil, nil
}

type MethodDexFundPledgeForVx struct {
}

func (md *MethodDexFundPledgeForVx) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForVx) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPledgeForVx) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundPledgeForVxGas, data)
}

func (md *MethodDexFundPledgeForVx) GetReceiveQuota() uint64 {
	return dexFundPledgeForVxReceiveGas
}

func (md *MethodDexFundPledgeForVx) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md MethodDexFundPledgeForVx) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundPledgeForVx)
	if err := ABIDexFund.UnpackMethod(param, MethodNameDexFundPledgeForVx, sendBlock.Data); err != nil {
		return []*ledger.AccountBlock{}, err
	}
	return handlePledgeAction(db, block, dex.PledgeForVx, param.ActionType, sendBlock.AccountAddress, param.Amount)
}

type MethodDexFundPledgeForVip struct {
}

func (md *MethodDexFundPledgeForVip) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForVip) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPledgeForVip) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundPledgeForVipGas, data)
}

func (md *MethodDexFundPledgeForVip) GetReceiveQuota() uint64 {
	return dexFundPledgeForVipReceiveGas
}

func (md *MethodDexFundPledgeForVip) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md MethodDexFundPledgeForVip) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundPledgeForVip)
	if err := ABIDexFund.UnpackMethod(param, MethodNameDexFundPledgeForVip, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	return handlePledgeAction(db, block, dex.PledgeForVip, param.ActionType, sendBlock.AccountAddress, dex.PledgeForVipAmount)
}

type MethodDexFundPledgeCallback struct {
}

func (md *MethodDexFundPledgeCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeCallback) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPledgeCallback) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundPledgeCallbackGas, data)
}

func (md *MethodDexFundPledgeCallback) GetReceiveQuota() uint64 {
	return dexFundPledgeCallbackReceiveGas
}

func (md *MethodDexFundPledgeCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidSourceAddressErr
	}
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), MethodNameDexFundPledgeCallback, block.Data)
}

func (md MethodDexFundPledgeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		originSendBlock *ledger.AccountBlock
		callbackParam   = new(dex.ParamDexFundPledgeCallBack)
	)
	if err = ABIDexFund.UnpackMethod(callbackParam, MethodNameDexFundPledgeCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if originSendBlock, err = GetOriginSendBlock(db, sendBlock.Hash); err != nil {
		panic(err)
	}
	pledgeParam := new(dex.ParamDexFundPledge)
	if err = cabi.ABIPledge.UnpackMethod(pledgeParam, cabi.MethodNameAgentPledge, originSendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if callbackParam.Success {
		if pledgeParam.PledgeType == dex.PledgeForVip {
			var pledgeVip *dex.PledgeVip
			if pledgeVip, err = dex.GetPledgeForVip(db, pledgeParam.PledgeAddress); err != nil {
				return handleReceiveErr(db, err)
			}
			if pledgeVip != nil {
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes + 1
				if err = dex.SavePledgeForVip(db, pledgeParam.PledgeAddress, pledgeVip); err != nil {
					return handleReceiveErr(db, err)
				}
				return doCancelPledge(db, block, pledgeParam.PledgeAddress, pledgeParam.PledgeType, originSendBlock.Amount)
			} else {
				pledgeVip = &dex.PledgeVip{}
				pledgeVip.Timestamp = dex.GetTimestampInt64(db)
				pledgeVip.PledgeTimes = 1
				if err = dex.SavePledgeForVip(db, pledgeParam.PledgeAddress, pledgeVip); err != nil {
					return handleReceiveErr(db, err)
				}
			}
		} else {
			pledgeAmount := dex.GetPledgeForVx(db, pledgeParam.PledgeAddress)
			pledgeAmount.Add(pledgeAmount, originSendBlock.Amount)
			if err = dex.SavePledgeForVx(db, pledgeParam.PledgeAddress, pledgeAmount); err != nil {
				return handleReceiveErr(db, err)
			}
		}
	} else {
		if sendBlock.Amount.Cmp(originSendBlock.Amount) != 0 {
			return handleReceiveErr(db, dex.InvalidAmountForPledgeCallbackErr)
		}
		if _, err = dex.DepositAccount(db, pledgeParam.PledgeAddress, ledger.ViteTokenId, sendBlock.Amount); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	return nil, nil
}

type MethodDexFundCancelPledgeCallback struct {
}

func (md *MethodDexFundCancelPledgeCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelPledgeCallback) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundCancelPledgeCallback) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundCancelPledgeCallbackGas, data)
}

func (md *MethodDexFundCancelPledgeCallback) GetReceiveQuota() uint64 {
	return dexFundCancelPledgeCallbackReceiveGas
}

func (md *MethodDexFundCancelPledgeCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidSourceAddressErr
	}
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), MethodNameDexFundCancelPledgeCallback, block.Data)
}

func (md MethodDexFundCancelPledgeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		originSendBlock *ledger.AccountBlock
		callbackParam   = new(dex.ParamDexFundPledgeCallBack)
	)
	if err = ABIDexFund.UnpackMethod(callbackParam, MethodNameDexFundCancelPledgeCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if originSendBlock, err = GetOriginSendBlock(db, sendBlock.Hash); err != nil {
		panic(err)
	}
	cancelPledgeParam := new(dex.ParamDexFundCancelPledge)
	if err = cabi.ABIPledge.UnpackMethod(cancelPledgeParam, cabi.MethodNameAgentCancelPledge, originSendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if callbackParam.Success {
		if sendBlock.Amount.Cmp(originSendBlock.Amount) != 0 {
			return handleReceiveErr(db, dex.InvalidAmountForPledgeCallbackErr)
		}
		if cancelPledgeParam.PledgeType == dex.PledgeForVip {
			var pledgeVip *dex.PledgeVip
			if pledgeVip, err = dex.GetPledgeForVip(db, cancelPledgeParam.PledgeAddress); err != nil {
				return handleReceiveErr(db, err)
			}
			if pledgeVip != nil {
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes - 1
				if pledgeVip.PledgeTimes == 0 {
					if err = dex.DeletePledgeForVip(db, cancelPledgeParam.PledgeAddress); err != nil {
						return handleReceiveErr(db, err)
					}
				} else {
					if err = dex.SavePledgeForVip(db, cancelPledgeParam.PledgeAddress, pledgeVip); err != nil {
						return handleReceiveErr(db, err)
					}
				}
			} else {
				return handleReceiveErr(db, dex.PledgeForVipNotExistsErr)
			}
		} else {
			pledgeAmount := dex.GetPledgeForVx(db, cancelPledgeParam.PledgeAddress)
			leaved := new(big.Int).Sub(pledgeAmount, sendBlock.Amount)
			if leaved.Sign() < 0 {
				return handleReceiveErr(db, dex.InvalidAmountForPledgeCallbackErr)
			} else if leaved.Sign() == 0 {
				if err = dex.DeletePledgeForVx(db, cancelPledgeParam.PledgeAddress); err != nil {
					return handleReceiveErr(db, err)
				}
			} else {
				if err = dex.SavePledgeForVx(db, cancelPledgeParam.PledgeAddress, leaved); err != nil {
					return handleReceiveErr(db, err)
				}
			}
		}
		if _, err = dex.DepositAccount(db, cancelPledgeParam.PledgeAddress, ledger.ViteTokenId, sendBlock.Amount); err != nil {
			return handleReceiveErr(db, err)
		}
	}
	return nil, nil
}

type MethodDexFundGetTokenInfoCallback struct {
}

func (md *MethodDexFundGetTokenInfoCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundGetTokenInfoCallback) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundGetTokenInfoCallback) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundGetTokenInfoCallbackGas, data)
}

func (md *MethodDexFundGetTokenInfoCallback) GetReceiveQuota() uint64 {
	return dexFundGetTokenInfoCallbackReceiveGas
}

func (md *MethodDexFundGetTokenInfoCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressMintage {
		return dex.InvalidSourceAddressErr
	}
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundGetTokenInfoCallback), MethodNameDexFundGetTokenInfoCallback, block.Data)
}

func (md MethodDexFundGetTokenInfoCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		originSendBlock *ledger.AccountBlock
		callbackParam   = new(dex.ParamDexFundGetTokenInfoCallback)
	)
	if err = ABIDexFund.UnpackMethod(callbackParam, MethodNameDexFundGetTokenInfoCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if originSendBlock, err = GetOriginSendBlock(db, sendBlock.Hash); err != nil {
		panic(err)
	}
	tradeTokenId := new(types.TokenTypeId)
	if err = cabi.ABIMintage.UnpackMethod(tradeTokenId, cabi.MethodNameGetTokenInfo, originSendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if callbackParam.Exist {
		if err = dex.OnGetTokenInfoSuccess(db, vm.ConsensusReader(), *tradeTokenId, callbackParam); err != nil {
			return handleReceiveErr(db, err)
		}
	} else {
		if refundBlocks, err := dex.OnGetTokenInfoFailed(db, *tradeTokenId); err != nil {
			return handleReceiveErr(db, err)
		} else {
			return refundBlocks, nil
		}
	}
	return nil, nil
}

type MethodDexFundConfigTimerAddress struct {
}

func (md *MethodDexFundConfigTimerAddress) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundConfigTimerAddress) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundConfigTimerAddress) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundConfigTimerAddressGas, data)
}

func (md *MethodDexFundConfigTimerAddress) GetReceiveQuota() uint64 {
	return dexFundConfigTimerAddressReceiveGas
}

func (md *MethodDexFundConfigTimerAddress) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundNotifyTime), MethodNameDexFundConfigTimerAddress, block.Data)
}

func (md MethodDexFundConfigTimerAddress) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err                     error
		ConfigTimerAddressParam = new(dex.ParamDexConfigTimerAddress)
	)
	if !dex.IsOwner(db, sendBlock.AccountAddress) {
		return handleReceiveErr(db, dex.OnlyOwnerAllowErr)
	}
	if err = ABIDexFund.UnpackMethod(ConfigTimerAddressParam, MethodNameDexFundConfigTimerAddress, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if err = dex.SetTimerAddress(db, ConfigTimerAddressParam.Address); err != nil {
		return handleReceiveErr(db, err)
	}
	return nil, nil
}

type MethodDexFundNotifyTime struct {
}

func (md *MethodDexFundNotifyTime) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNotifyTime) GetRefundData() ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNotifyTime) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNotifyTimeGas, data)
}

func (md *MethodDexFundNotifyTime) GetReceiveQuota() uint64 {
	return dexFundNotifyTimeReceiveGas
}

func (md *MethodDexFundNotifyTime) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return ABIDexFund.UnpackMethod(new(dex.ParamDexFundNotifyTime), MethodNameDexFundNotifyTime, block.Data)
}

func (md MethodDexFundNotifyTime) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		notifyTimeParam = new(dex.ParamDexFundNotifyTime)
	)
	if !dex.ValidTimerAddress(db, sendBlock.AccountAddress) {
		return handleReceiveErr(db, dex.InvalidSourceAddressErr)
	}
	if err = ABIDexFund.UnpackMethod(notifyTimeParam, MethodNameDexFundNotifyTime, sendBlock.Data); err != nil {
		return handleReceiveErr(db, err)
	}
	if err = dex.SetTimerTimestamp(db, notifyTimeParam.Timestamp); err != nil {
		return handleReceiveErr(db, err)
	}
	return nil, nil
}

func handlePledgeAction(db vm_db.VmDb, block *ledger.AccountBlock, pledgeType uint8, actionType int8, address types.Address, amount *big.Int) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if actionType == dex.Pledge {
		if methodData, err = dex.PledgeRequest(db, address, pledgeType, amount); err != nil {
			return []*ledger.AccountBlock{}, err
		} else {
			return []*ledger.AccountBlock{
				{
					AccountAddress: block.AccountAddress,
					ToAddress:      types.AddressPledge,
					BlockType:      ledger.BlockTypeSendCall,
					Amount:         amount,
					TokenId:        ledger.ViteTokenId,
					Data:           methodData,
				},
			}, nil
		}
	} else {
		return doCancelPledge(db, block, address, pledgeType, amount)
	}
}

func doCancelPledge(db vm_db.VmDb, block *ledger.AccountBlock, address types.Address, pledgeType uint8, amount *big.Int) ([]*ledger.AccountBlock, error) {
	var (
		methodData []byte
		err        error
	)
	if methodData, err = dex.CancelPledgeRequest(db, address, pledgeType, amount); err != nil {
		return []*ledger.AccountBlock{}, err
	} else {
		return []*ledger.AccountBlock{
			{
				AccountAddress: block.AccountAddress,
				ToAddress:      types.AddressPledge,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           methodData,
			},
		}, nil
	}
}

func handleNewOrderFail(db vm_db.VmDb, orderInfo *dexproto.OrderInfo, errCode int) ([]*ledger.AccountBlock, error) {
	orderInfo.Order.Status = dex.NewFailed
	dex.EmitOrderFailLog(db, orderInfo, errCode)
	return nil, nil
}

func handleReceiveErr(db vm_db.VmDb, err error) ([]*ledger.AccountBlock, error) {
	dex.EmitErrLog(db, err)
	return nil, nil
}
