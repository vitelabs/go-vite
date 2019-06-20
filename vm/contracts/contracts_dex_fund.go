package contracts

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type MethodDexFundUserDeposit struct {
}

func (md *MethodDexFundUserDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserDeposit) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	account := dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount)
	// must do after account updated by deposit
	if sendBlock.TokenId == dex.VxTokenId {
		dex.OnDepositVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, sendBlock.Amount, account)
	}
	return nil, nil
}

type MethodDexFundUserWithdraw struct {
}

func (md *MethodDexFundUserWithdraw) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserWithdraw) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundUserWithdraw, block.Data); err != nil {
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
		ok      bool
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundUserWithdraw, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if dexFund, ok = dex.GetUserFund(db, sendBlock.AccountAddress); !ok {
		return handleReceiveErr(dex.ExceedFundAvailableErr)
	}
	account, _ := dex.GetAccountByTokeIdFromFund(dexFund, param.Token)
	available := dex.SubBigInt(account.Available, param.Amount.Bytes())
	if available.Sign() < 0 {
		return handleReceiveErr(dex.ExceedFundAvailableErr)
	}
	account.Available = available.Bytes()
	// must do after account updated by withdraw
	if param.Token == dex.VxTokenId {
		dex.OnWithdrawVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, param.Amount, account)
	}
	dex.SaveUserFund(db, sendBlock.AccountAddress, dexFund)
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

type MethodDexFundNewMarket struct {
}

func (md *MethodDexFundNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewMarket) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewMarket, block.Data); err != nil {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewMarket, sendBlock.Data); err != nil {
		return nil, err
	}
	if _, ok := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); ok {
		return nil, dex.TradeMarketExistsErr
	}
	marketInfo := &dex.MarketInfo{}
	if err = dex.RenderMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken, nil, &sendBlock.AccountAddress); err != nil {
		return nil, err
	}
	exceedAmount := new(big.Int).Sub(sendBlock.Amount, dex.NewMarketFeeAmount)
	if exceedAmount.Sign() > 0 {
		dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, exceedAmount)
	}
	if marketInfo.Valid {
		appendBlock := dex.OnNewMarketValid(db, vm.ConsensusReader(), marketInfo, param.TradeToken, param.QuoteToken, &sendBlock.AccountAddress)
		return []*ledger.AccountBlock{appendBlock}, nil
	} else {
		getTokenInfoData := dex.OnNewMarketPending(db, param, marketInfo)
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressMintage,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           getTokenInfoData,
			},
		}, nil
	}
	return nil, nil
}

type MethodDexFundNewOrder struct {
}

func (md *MethodDexFundNewOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewOrder) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewOrder) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewOrderGas, data)
}

func (md *MethodDexFundNewOrder) GetReceiveQuota() uint64 {
	return dexFundNewOrderReceiveGas
}

func (md *MethodDexFundNewOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	param := new(dex.ParamDexFundNewOrder)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewOrder, block.Data); err != nil {
		return err
	}
	err = dex.PreCheckOrderParam(param)
	return
}

func (md *MethodDexFundNewOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		dexFund        = &dex.UserFund{}
		tradeBlockData []byte
		err            error
		orderInfoBytes []byte
		marketInfo     *dex.MarketInfo
		ok             bool
	)
	param := new(dex.ParamDexFundNewOrder)
	cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewOrder, sendBlock.Data)
	order := &dex.Order{}
	if marketInfo, err = dex.RenderOrder(order, param, db, sendBlock.AccountAddress); err != nil {
		return handleReceiveErr(err)
	}
	if dexFund, ok = dex.GetUserFund(db, sendBlock.AccountAddress); !ok {
		return handleReceiveErr(dex.ExceedFundAvailableErr)
	}
	if err = dex.CheckAndLockFundForNewOrder(dexFund, order, marketInfo); err != nil {
		return handleReceiveErr(err)
	}
	dex.SaveUserFund(db, sendBlock.AccountAddress, dexFund)
	if orderInfoBytes, err = order.Serialize(); err != nil {
		panic(err)
	}
	if tradeBlockData, err = cabi.ABIDexTrade.PackMethod(cabi.MethodNameDexTradeNewOrder, orderInfoBytes); err != nil {
		panic(err)
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

func (md *MethodDexFundSettleOrders) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
		return dex.InvalidSourceAddressErr
	}
	param := new(dex.ParamDexSerializedData)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundSettleOrders, block.Data); err != nil {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundSettleOrders, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return handleReceiveErr(err)
	}
	if marketInfo, ok := dex.GetMarketInfoByTokens(db, settleActions.TradeToken, settleActions.QuoteToken); !ok {
		return handleReceiveErr(dex.TradeMarketNotExistsErr)
	} else {
		for _, fundAction := range settleActions.FundActions {
			if err = dex.DoSettleFund(db, vm.ConsensusReader(), fundAction, marketInfo); err != nil {
				return handleReceiveErr(err)
			}
		}
		if len(settleActions.FeeActions) > 0 {
			dex.SettleFees(db, vm.ConsensusReader(), marketInfo.AllowMine, marketInfo.QuoteToken, marketInfo.QuoteTokenDecimals, marketInfo.QuoteTokenType, settleActions.FeeActions, nil, nil)
			dex.SettleBrokerFeeSum(db, vm.ConsensusReader(), settleActions.FeeActions, marketInfo)
		}
		return nil, nil
	}
}

type MethodDexFundFeeDividend struct {
}

func (md *MethodDexFundFeeDividend) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundFeeDividend) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundFeeDividend) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundFeeDividendGas, data)
}

func (md *MethodDexFundFeeDividend) GetReceiveQuota() uint64 {
	return dexFundFeeDividendReceiveGas
}

func (md *MethodDexFundFeeDividend) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundDividend), cabi.MethodNameDexFundFeeDividend, block.Data)
}

func (md MethodDexFundFeeDividend) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err error
	)
	if !dex.IsOwner(db, sendBlock.AccountAddress) {
		return handleReceiveErr(dex.OnlyOwnerAllowErr)
	}
	param := new(dex.ParamDexFundDividend)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundFeeDividend, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if param.PeriodId >= dex.GetCurrentPeriodId(db, vm.ConsensusReader()) {
		return handleReceiveErr(fmt.Errorf("dividend periodId not before current periodId"))
	}
	if lastPeriodId := dex.GetLastFeeDividendPeriodId(db); lastPeriodId > 0 && param.PeriodId != lastPeriodId+1 {
		return handleReceiveErr(fmt.Errorf("fee dividend period id not equals to expected id %d", lastPeriodId+1))
	}
	if err = dex.DoDivideFees(db, param.PeriodId); err != nil {
		return handleReceiveErr(err)
	} else {
		dex.SaveLastFeeDividendPeriodId(db, param.PeriodId)
	}
	return nil, nil
}

type MethodDexFundMinedVxDividend struct {
}

func (md *MethodDexFundMinedVxDividend) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundMinedVxDividend) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundMinedVxDividend) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundMinedVxDividendGas, data)
}

func (md *MethodDexFundMinedVxDividend) GetReceiveQuota() uint64 {
	return dexFundMinedVxDividendReceiveGas
}

func (md *MethodDexFundMinedVxDividend) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundDividend), cabi.MethodNameDexFundMinedVxDividend, block.Data)
}

func (md MethodDexFundMinedVxDividend) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		vxBalance *big.Int
		err       error
	)
	if !dex.IsOwner(db, sendBlock.AccountAddress) {
		return handleReceiveErr(dex.OnlyOwnerAllowErr)
	}
	param := new(dex.ParamDexFundDividend)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundMinedVxDividend, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if param.PeriodId >= dex.GetCurrentPeriodId(db, vm.ConsensusReader()) {
		return handleReceiveErr(fmt.Errorf("specified periodId for mined vx not before current periodId"))
	}
	if lastMinedVxPeriodId := dex.GetLastMinedVxPeriodId(db); lastMinedVxPeriodId > 0 && param.PeriodId != lastMinedVxPeriodId+1 {
		return handleReceiveErr(fmt.Errorf("mined vx period id not equals to expected id %d", lastMinedVxPeriodId+1))
	}
	vxBalance = dex.GetVxBalance(db)
	if amtForFeePerMarket, amtForMaker, amtForMaintainer, amtForPledge, vxLeaved, success := dex.GetMindedVxAmt(db, param.PeriodId, vxBalance); !success {
		return handleReceiveErr(fmt.Errorf("no vx available for mine"))
	} else {
		if err = dex.DoMineVxForFee(db, param.PeriodId, amtForFeePerMarket); err != nil {
			return handleReceiveErr(err)
		}
		if err = dex.DoMineVxForPledge(db, param.PeriodId, amtForPledge); err != nil {
			return handleReceiveErr(err)
		}
		if err = dex.DoMineVxForMakerMineAndMaintainer(db, amtForMaker, amtForMaintainer); err != nil {
			return handleReceiveErr(err)
		}
		dex.SaveVxBalance(db, vxLeaved)
	}
	dex.SaveLastMinedVxPeriodId(db, param.PeriodId)
	return nil, nil
}

type MethodDexFundPledgeForVx struct {
}

func (md *MethodDexFundPledgeForVx) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForVx) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForVx, block.Data); err != nil {
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
	if err := cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForVx, sendBlock.Data); err != nil {
		return []*ledger.AccountBlock{}, err
	}
	return dex.HandlePledgeAction(db, block, dex.PledgeForVx, param.ActionType, sendBlock.AccountAddress, param.Amount)
}

type MethodDexFundPledgeForVip struct {
}

func (md *MethodDexFundPledgeForVip) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForVip) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForVip, block.Data); err != nil {
		return err
	}
	if param.ActionType != dex.Pledge && param.ActionType != dex.CancelPledge {
		return dex.InvalidPledgeActionTypeErr
	}
	return nil
}

func (md MethodDexFundPledgeForVip) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundPledgeForVip)
	if err := cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForVip, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	return dex.HandlePledgeAction(db, block, dex.PledgeForVip, param.ActionType, sendBlock.AccountAddress, dex.PledgeForVipAmount)
}

type MethodDexFundPledgeCallback struct {
}

func (md *MethodDexFundPledgeCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeCallback) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), cabi.MethodNameDexFundPledgeCallback, block.Data)
}

func (md MethodDexFundPledgeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err           error
		callbackParam = new(dex.ParamDexFundPledgeCallBack)
	)
	if err = cabi.ABIDexFund.UnpackMethod(callbackParam, cabi.MethodNameDexFundPledgeCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if callbackParam.Success {
		if callbackParam.Bid == dex.PledgeForVip {
			if pledgeVip, ok := dex.GetPledgeForVip(db, callbackParam.PledgeAddress); ok {
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes + 1
				dex.SavePledgeForVip(db, callbackParam.PledgeAddress, pledgeVip)
				// duplicate pledge for vip, cancel pledge
				return dex.DoCancelPledge(db, block, callbackParam.PledgeAddress, callbackParam.Bid, callbackParam.Amount)
			} else {
				pledgeVip.Timestamp = dex.GetTimestampInt64(db)
				pledgeVip.PledgeTimes = 1
				dex.SavePledgeForVip(db, callbackParam.PledgeAddress, pledgeVip)
			}
		} else {
			pledgeAmount := dex.GetPledgeForVx(db, callbackParam.PledgeAddress)
			pledgeAmount.Add(pledgeAmount, callbackParam.Amount)
			dex.SavePledgeForVx(db, callbackParam.PledgeAddress, pledgeAmount)
			dex.OnPledgeForVxSuccess(db, vm.ConsensusReader(), callbackParam.PledgeAddress, callbackParam.Amount, pledgeAmount)
		}
	} else {
		if callbackParam.Bid == dex.PledgeForVip {
			if dex.PledgeForVipAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
		} else {
			if callbackParam.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
		}
		dex.DepositAccount(db, callbackParam.PledgeAddress, ledger.ViteTokenId, sendBlock.Amount)
	}
	return nil, nil
}

type MethodDexFundCancelPledgeCallback struct {
}

func (md *MethodDexFundCancelPledgeCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelPledgeCallback) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), cabi.MethodNameDexFundCancelPledgeCallback, block.Data)
}

func (md MethodDexFundCancelPledgeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err               error
		cancelPledgeParam = new(dex.ParamDexFundPledgeCallBack)
	)
	if err = cabi.ABIDexFund.UnpackMethod(cancelPledgeParam, cabi.MethodNameDexFundCancelPledgeCallback, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if cancelPledgeParam.Success {
		if cancelPledgeParam.Bid == dex.PledgeForVip {
			if dex.PledgeForVipAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
			if pledgeVip, ok := dex.GetPledgeForVip(db, cancelPledgeParam.PledgeAddress); ok {
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes - 1
				if pledgeVip.PledgeTimes == 0 {
					dex.DeletePledgeForVip(db, cancelPledgeParam.PledgeAddress)
				} else {
					dex.SavePledgeForVip(db, cancelPledgeParam.PledgeAddress, pledgeVip)
				}
			} else {
				return handleReceiveErr(dex.PledgeForVipNotExistsErr)
			}
		} else {
			if cancelPledgeParam.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
			pledgeAmount := dex.GetPledgeForVx(db, cancelPledgeParam.PledgeAddress)
			leaved := new(big.Int).Sub(pledgeAmount, sendBlock.Amount)
			if leaved.Sign() < 0 {
				return handleReceiveErr(dex.InvalidAmountForPledgeCallbackErr)
			} else if leaved.Sign() == 0 {
				dex.DeletePledgeForVx(db, cancelPledgeParam.PledgeAddress)
			} else {
				dex.SavePledgeForVx(db, cancelPledgeParam.PledgeAddress, leaved)
			}
			dex.OnCancelPledgeForVxSuccess(db, vm.ConsensusReader(), cancelPledgeParam.PledgeAddress, sendBlock.Amount, leaved)
		}
		dex.DepositAccount(db, cancelPledgeParam.PledgeAddress, ledger.ViteTokenId, sendBlock.Amount)
	}
	return nil, nil
}

type MethodDexFundGetTokenInfoCallback struct {
}

func (md *MethodDexFundGetTokenInfoCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundGetTokenInfoCallback) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
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
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundGetTokenInfoCallback), cabi.MethodNameDexFundGetTokenInfoCallback, block.Data)
}

func (md MethodDexFundGetTokenInfoCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var callbackParam = new(dex.ParamDexFundGetTokenInfoCallback)
	cabi.ABIDexFund.UnpackMethod(callbackParam, cabi.MethodNameDexFundGetTokenInfoCallback, sendBlock.Data)
	switch callbackParam.Bid {
	case dex.GetTokenForNewMarket:
		if callbackParam.Exist {
			if appendBlocks, err := dex.OnNewMarketGetTokenInfoSuccess(db, vm.ConsensusReader(), callbackParam.TokenId, callbackParam); err != nil {
				return handleReceiveErr(err)
			} else {
				return appendBlocks, nil
			}
		} else {
			if refundBlocks, err := dex.OnNewMarketGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleReceiveErr(err)
			} else {
				return refundBlocks, nil
			}
		}
	case dex.GetTokenForSetQuote:
		if callbackParam.Exist {
			if err := dex.OnSetQuoteGetTokenInfoSuccess(db, callbackParam); err != nil {
				return handleReceiveErr(err)
			}
		} else {
			if err := dex.OnSetQuoteGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleReceiveErr(err)
			}
		}
	case dex.GetTokenForTransferOwner:
		if callbackParam.Exist {
			if err := dex.OnTransferOwnerGetTokenInfoSuccess(db, callbackParam); err != nil {
				return handleReceiveErr(err)
			}
		} else {
			if err := dex.OnTransferOwnerGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleReceiveErr(err)
			}
		}
	}
	return nil, nil
}

type MethodDexFundOwnerConfig struct {
}

func (md *MethodDexFundOwnerConfig) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundOwnerConfig) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundOwnerConfig) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(DexFundOwnerConfigGas, data)
}

func (md *MethodDexFundOwnerConfig) GetReceiveQuota() uint64 {
	return DexFundOwnerConfigReceiveGas
}

func (md *MethodDexFundOwnerConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundOwnerConfig), cabi.MethodNameDexFundOwnerConfig, block.Data)
}

func (md MethodDexFundOwnerConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexFundOwnerConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundOwnerConfig, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigOwner) {
			dex.SetOwner(db, param.Owner)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTimerAddress) {
			dex.SetTimerAddress(db, param.TimerAddress)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigStopViteX) {
			dex.SaveViteXStopped(db, param.StopViteX)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMakerMineProxy) {
			dex.SaveMakerMineProxy(db, param.MakerMineProxy)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMaintainer) {
			dex.SaveMaintainer(db, param.Maintainer)
		}
	} else {
		return handleReceiveErr(dex.OnlyOwnerAllowErr)
	}
	return nil, nil
}

type MethodDexFundOwnerConfigTrade struct {
}

func (md *MethodDexFundOwnerConfigTrade) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundOwnerConfigTrade) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundOwnerConfigTrade) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(DexFundOwnerConfigTradeGas, data)
}

func (md *MethodDexFundOwnerConfigTrade) GetReceiveQuota() uint64 {
	return DexFundOwnerConfigTradeReceiveGas
}

func (md *MethodDexFundOwnerConfigTrade) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundOwnerConfigTrade), cabi.MethodNameDexFundOwnerConfigTrade, block.Data)
}

func (md MethodDexFundOwnerConfigTrade) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexFundOwnerConfigTrade)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundOwnerConfigTrade, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMineMarket) {
			if marketInfo, ok := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); ok && marketInfo.Valid {
				if param.AllowMine != marketInfo.AllowMine {
					marketInfo.AllowMine = param.AllowMine
					dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
					dex.AddMarketEvent(db, marketInfo)
				} else {
					if marketInfo.AllowMine {
						return handleReceiveErr(dex.TradeMarketAllowMineErr)
					} else {
						return handleReceiveErr(dex.TradeMarketNotAllowMineErr)
					}
				}
			} else {
				return handleReceiveErr(dex.TradeMarketNotExistsErr)
			}
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigNewQuoteToken) {
			if tokenInfo, ok := dex.GetTokenInfo(db, param.NewQuoteToken); !ok {
				getTokenInfoData := dex.OnSetQuoteTokenPending(db, param.NewQuoteToken, param.QuoteTokenType)
				return []*ledger.AccountBlock{
					{
						AccountAddress: types.AddressDexFund,
						ToAddress:      types.AddressMintage,
						BlockType:      ledger.BlockTypeSendCall,
						TokenId:        ledger.ViteTokenId,
						Amount:         big.NewInt(0),
						Data:           getTokenInfoData,
					},
				}, nil
			} else {
				if tokenInfo.QuoteTokenType > 0 {
					return handleReceiveErr(dex.AlreadyQuoteType)
				} else {
					tokenInfo.QuoteTokenType = int32(param.QuoteTokenType)
					dex.SaveTokenInfo(db, param.NewQuoteToken, tokenInfo)
					dex.AddTokenEvent(db, tokenInfo)
				}
			}
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTradeThreshold) {
			dex.SaveTradeThreshold(db, param.TokenType4TradeThr, param.TradeThreshold)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMineThreshold) {
			dex.SaveMineThreshold(db, param.TokenType4MineThr, param.MineThreshold)
		}
	} else {
		return handleReceiveErr(dex.OnlyOwnerAllowErr)
	}
	return nil, nil
}


type MethodDexFundMarketOwnerConfig struct {
}

func (md *MethodDexFundMarketOwnerConfig) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundMarketOwnerConfig) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundMarketOwnerConfig) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(DexFundMarketOwnerConfigGas, data)
}

func (md *MethodDexFundMarketOwnerConfig) GetReceiveQuota() uint64 {
	return DexFundMarketOwnerConfigReceiveGas
}

func (md *MethodDexFundMarketOwnerConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundMarketOwnerConfig), cabi.MethodNameDexFundMarketOwnerConfig, block.Data)
}

func (md MethodDexFundMarketOwnerConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err        error
		marketInfo *dex.MarketInfo
		ok         bool
	)
	var param = new(dex.ParamDexFundMarketOwnerConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundMarketOwnerConfig, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if marketInfo, ok = dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); !ok || !marketInfo.Valid {
		return handleReceiveErr(dex.TradeMarketNotExistsErr)
	}
	if bytes.Equal(sendBlock.AccountAddress.Bytes(), marketInfo.Owner) {
		if param.OperationCode == 0 {
			return nil, nil
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerTransferOwner) {
			marketInfo.Owner = param.Owner.Bytes()
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigTakerRate) {
			if !dex.ValidBrokerFeeRate(param.TakerRate) {
				return handleReceiveErr(dex.InvalidBrokerFeeRateErr)
			}
			marketInfo.TakerBrokerFeeRate = param.TakerRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigMakerRate) {
			if !dex.ValidBrokerFeeRate(param.MakerRate) {
				return handleReceiveErr(dex.InvalidBrokerFeeRateErr)
			}
			marketInfo.MakerBrokerFeeRate = param.MakerRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerStopMarket) {
			marketInfo.Stopped = param.StopMarket
		}
		dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
		dex.AddMarketEvent(db, marketInfo)
	} else {
		return handleReceiveErr(dex.OnlyOwnerAllowErr)
	}
	return nil, nil
}

type MethodDexFundTransferTokenOwner struct {
}

func (md *MethodDexFundTransferTokenOwner) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundTransferTokenOwner) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundTransferTokenOwner) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundTransferTokenOwnerGas, data)
}

func (md *MethodDexFundTransferTokenOwner) GetReceiveQuota() uint64 {
	return dexFundTransferTokenOwnerReceiveGas
}

func (md *MethodDexFundTransferTokenOwner) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundTransferTokenOwner), cabi.MethodNameDexFundTransferTokenOwner, block.Data)
}

func (md MethodDexFundTransferTokenOwner) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err   error
		param = new(dex.ParamDexFundTransferTokenOwner)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundTransferTokenOwner, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if tokenInfo, ok := dex.GetTokenInfo(db, param.Token); ok {
		if bytes.Equal(tokenInfo.Owner, sendBlock.AccountAddress.Bytes()) {
			tokenInfo.Owner = param.Owner.Bytes()
			dex.SaveTokenInfo(db, param.Token, tokenInfo)
			dex.AddTokenEvent(db, tokenInfo)
		} else {
			return handleReceiveErr(dex.OnlyOwnerAllowErr)
		}
	} else {
		getTokenInfoData := dex.OnTransferTokenOwnerPending(db, param.Token, sendBlock.AccountAddress, param.Owner)
		return []*ledger.AccountBlock{
			{
				AccountAddress: types.AddressDexFund,
				ToAddress:      types.AddressMintage,
				BlockType:      ledger.BlockTypeSendCall,
				TokenId:        ledger.ViteTokenId,
				Amount:         big.NewInt(0),
				Data:           getTokenInfoData,
			},
		}, nil
	}
	return nil, nil
}

type MethodDexFundNotifyTime struct {
}

func (md *MethodDexFundNotifyTime) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNotifyTime) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNotifyTime) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNotifyTimeGas, data)
}

func (md *MethodDexFundNotifyTime) GetReceiveQuota() uint64 {
	return dexFundNotifyTimeReceiveGas
}

func (md *MethodDexFundNotifyTime) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundNotifyTime), cabi.MethodNameDexFundNotifyTime, block.Data)
}

func (md MethodDexFundNotifyTime) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		notifyTimeParam = new(dex.ParamDexFundNotifyTime)
	)
	if !dex.ValidTimerAddress(db, sendBlock.AccountAddress) {
		return handleReceiveErr(dex.InvalidSourceAddressErr)
	}
	if err = cabi.ABIDexFund.UnpackMethod(notifyTimeParam, cabi.MethodNameDexFundNotifyTime, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if err = dex.SetTimerTimestamp(db, notifyTimeParam.Timestamp); err != nil {
		return handleReceiveErr(err)
	}
	return nil, nil
}

type MethodDexFundNewInviter struct {
}

func (md *MethodDexFundNewInviter) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewInviter) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewInviter) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundNewInviterGas, data)
}

func (md *MethodDexFundNewInviter) GetReceiveQuota() uint64 {
	return dexFundNewInviterReceiveGas
}

func (md *MethodDexFundNewInviter) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Cmp(dex.NewInviterFeeAmount) < 0 {
		return dex.InvalidInviterFeeAmountErr
	}
	return nil
}

func (md MethodDexFundNewInviter) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	if code := dex.GetCodeByInviter(db, sendBlock.AccountAddress); code > 0 {
		return nil, dex.AlreadyIsInviterErr
	}
	exceedAmount := new(big.Int).Sub(sendBlock.Amount, dex.NewInviterFeeAmount)
	if exceedAmount.Sign() > 0 {
		dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, exceedAmount)
	}
	dex.SettleFeesWithTokenId(db, vm.ConsensusReader(), true, ledger.ViteTokenId, dex.ViteTokenTypeInfo.Decimals, dex.ViteTokenType, nil, dex.NewInviterFeeAmount, nil)
	if inviteCode := dex.NewInviteCode(db, block.PrevHash); inviteCode == 0 {
		return nil, dex.NewInviteCodeFailErr
	} else {
		dex.SaveCodeByInviter(db, sendBlock.AccountAddress, inviteCode)
		dex.SaveInviterByCode(db, sendBlock.AccountAddress, inviteCode)
	}
	return nil, nil
}

type MethodDexFundBindInviteCode struct {
}

func (md *MethodDexFundBindInviteCode) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundBindInviteCode) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundBindInviteCode) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(dexFundBindInviteCodeGas, data)
}

func (md *MethodDexFundBindInviteCode) GetReceiveQuota() uint64 {
	return dexFundBindInviteCodeReceiveGas
}

func (md *MethodDexFundBindInviteCode) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if err := cabi.ABIDexFund.UnpackMethod(new(string), cabi.MethodNameDexFundBindInviteCode, block.Data); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundBindInviteCode) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		inviter *types.Address
		code    uint32
		err     error
	)
	if err = cabi.ABIDexFund.UnpackMethod(&code, cabi.MethodNameDexFundBindInviteCode, sendBlock.Data); err != nil {
		return handleReceiveErr(err)
	}
	if _, err = dex.GetInviterByInvitee(db, sendBlock.AccountAddress); err != dex.NotBindInviterErr {
		if err == nil {
			return handleReceiveErr(dex.AlreadyBindInviterErr)
		} else {
			return handleReceiveErr(err)
		}
	}
	if inviter, err = dex.GetInviterByCode(db, code); err != nil {
		return handleReceiveErr(err)
	}
	dex.SaveInviterByInvitee(db, sendBlock.AccountAddress, *inviter)
	return nil, nil
}

func handleReceiveErr(err error) ([]*ledger.AccountBlock, error) {
	return nil, err
}
