package contracts

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var fundLogger = log15.New("module", "dex_fund")

type MethodDexFundUserDeposit struct {
}

func (md *MethodDexFundUserDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundUserDeposit) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundUserDeposit) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(util.TxGas, data)
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
	return util.TotalGasCost(util.TxGas, data)
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
		acc *dexproto.Account
		err error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundUserWithdraw, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundUserWithdraw, err, sendBlock)
	}
	if acc, err = dex.SubUserFund(db, sendBlock.AccountAddress, param.Token.Bytes(), param.Amount); err != nil {
		handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundUserWithdraw, err, sendBlock)
	} else {
		if param.Token == dex.VxTokenId {
			dex.OnWithdrawVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, param.Amount, acc)
		}
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

type MethodDexFundNewMarket struct {
}

func (md *MethodDexFundNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewMarket) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewMarket) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(util.TxGas, data)
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
	if err = dex.CheckMarketParam(param, block.TokenId); err != nil {
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
	if mk, ok := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); ok && mk.Valid { // if mk not valid, overwrite old marketInfo with new
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewMarket, dex.TradeMarketExistsErr, sendBlock)
	}
	marketInfo := &dex.MarketInfo{}
	if err = dex.RenderMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken, nil, &sendBlock.AccountAddress); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewMarket, err, sendBlock)
	}
	if marketInfo.Valid {
		if appendBlocks, err := dex.OnNewMarketValid(db, vm.ConsensusReader(), marketInfo, param.TradeToken, param.QuoteToken, &sendBlock.AccountAddress); err == nil {
			return appendBlocks, nil
		} else {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewMarket, err, sendBlock)
		}
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewOrder, err, sendBlock)
	}
	if dexFund, ok = dex.GetUserFund(db, sendBlock.AccountAddress); !ok {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewOrder, dex.ExceedFundAvailableErr, sendBlock)
	}
	if err = dex.CheckAndLockFundForNewOrder(dexFund, order, marketInfo); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewOrder, err, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleOrders, err, sendBlock)
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleOrders, err, sendBlock)
	}
	if marketInfo, ok := dex.GetMarketInfoByTokens(db, settleActions.TradeToken, settleActions.QuoteToken); !ok {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleOrders, dex.TradeMarketNotExistsErr, sendBlock)
	} else {
		for _, fundAction := range settleActions.FundActions {
			if err = dex.DoSettleFund(db, vm.ConsensusReader(), fundAction, marketInfo); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleOrders, err, sendBlock)
			}
		}
		if len(settleActions.FeeActions) > 0 {
			dex.SettleFees(db, vm.ConsensusReader(), marketInfo.AllowMine, marketInfo.QuoteToken, marketInfo.QuoteTokenDecimals, marketInfo.QuoteTokenType, settleActions.FeeActions, nil, nil)
			dex.SettleBrokerFeeSum(db, vm.ConsensusReader(), settleActions.FeeActions, marketInfo)
		}
		return nil, nil
	}
}

type MethodDexFundPeriodJob struct {
}

func (md *MethodDexFundPeriodJob) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPeriodJob) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPeriodJob) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(util.TxGas, data)
}

func (md *MethodDexFundPeriodJob) GetReceiveQuota() uint64 {
	return dexFundPeriodJobReceiveGas
}

func (md *MethodDexFundPeriodJob) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexPeriodJob), cabi.MethodNameDexFundPeriodJob, block.Data)
}

func (md MethodDexFundPeriodJob) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err error
	)
	if !dex.ValidTriggerAddress(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamDexPeriodJob)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPeriodJob, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, err, sendBlock)
	}
	if param.PeriodId >= dex.GetCurrentPeriodId(db, vm.ConsensusReader()) {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, fmt.Errorf("job periodId for biz %d not before current periodId", param.BizType), sendBlock)
	}
	if lastPeriodId := dex.GetLastJobPeriodIdByBizType(db, param.BizType); lastPeriodId > 0 && param.PeriodId != lastPeriodId+1 {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, fmt.Errorf("job periodId for biz %d  not equals to expected id %d", param.BizType, lastPeriodId+1), sendBlock)
	}
	dex.AddPeriodWithBizEvent(db, param.PeriodId, param.BizType)
	if param.BizType <= dex.BrokerFeeDividendJob {
		switch param.BizType {
		case dex.FeeDividendJob:
			err = dex.DoDivideFees(db, param.PeriodId)
		case dex.BrokerFeeDividendJob:
			err = dex.DoDivideBrokerFees(db, param.PeriodId)
		}
		if err != nil {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, err, sendBlock)
		}
	} else {
		var (
			vxPool, amount, vxPoolLeaved *big.Int
			amtForItems                  map[int32]*big.Int
			success                      bool
		)
		vxPool = dex.GetVxMinePool(db)
		switch param.BizType {
		case dex.MineVxForFeeJob:
			if amtForItems, vxPoolLeaved, success = dex.GetVxAmountsForEqualItems(db, param.PeriodId, vxPool, dex.RateSumForFeeMine, dex.ViteTokenType, dex.UsdTokenType); success {
				if err = dex.DoMineVxForFee(db, vm.ConsensusReader(), param.PeriodId, amtForItems); err != nil {
					return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, fmt.Errorf("no vx available on mine for fee"), sendBlock)
			}
		case dex.MineVxForPledgeJob:
			if amount, vxPoolLeaved, success = dex.GetVxAmountToMine(db, param.PeriodId, vxPool, dex.RateForPledgeMine); success {
				if err = dex.DoMineVxForPledge(db, vm.ConsensusReader(), param.PeriodId, amount); err != nil {
					return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, fmt.Errorf("no vx available on mine for pledge"), sendBlock)
			}
		case dex.MineVxForMakerAndMaintainerJob:
			if amtForItems, vxPoolLeaved, success = dex.GetVxAmountsForEqualItems(db, param.PeriodId, vxPool, dex.RateSumForMakerAndMaintainerMine, dex.MineForMaker, dex.MineForMaintainer); success {
				if err = dex.DoMineVxForMakerMineAndMaintainer(db, param.PeriodId, vm.ConsensusReader(), amtForItems); err != nil {
					return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, fmt.Errorf("no vx available on mine for maker and maintainer"), sendBlock)
			}
		}
		dex.SaveVxMinePool(db, vxPoolLeaved)
	}
	dex.SaveLastJobPeriodIdByBizType(db, param.PeriodId, param.BizType)
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
	return util.TotalGasCost(util.TxGas, data)
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
	if appendBlocks, err := dex.HandlePledgeAction(db, block, dex.PledgeForVx, param.ActionType, sendBlock.AccountAddress, param.Amount, nodeConfig.params.PledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeForVx, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeForVip, err, sendBlock)
	}
	if appendBlocks, err := dex.HandlePledgeAction(db, block, dex.PledgeForVip, param.ActionType, sendBlock.AccountAddress, dex.PledgeForVipAmount, nodeConfig.params.ViteXVipPledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeForVip, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeCallback, err, sendBlock)
	}
	if callbackParam.Success {
		if callbackParam.Bid == dex.PledgeForVip {
			if pledgeVip, ok := dex.GetPledgeForVip(db, callbackParam.PledgeAddress); ok { //duplicate pledge for vip
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundCancelPledgeCallback, err, sendBlock)
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
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundCancelPledgeCallback, dex.PledgeForVipNotExistsErr, sendBlock)
			}
		} else {
			if cancelPledgeParam.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
			pledgeAmount := dex.GetPledgeForVx(db, cancelPledgeParam.PledgeAddress)
			leaved := new(big.Int).Sub(pledgeAmount, sendBlock.Amount)
			if leaved.Sign() < 0 {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundCancelPledgeCallback, dex.InvalidAmountForPledgeCallbackErr, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundGetTokenInfoCallback, err, sendBlock)
			} else {
				return appendBlocks, nil
			}
		} else {
			if err := dex.OnNewMarketGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundGetTokenInfoCallback, err, sendBlock)
			}
		}
	case dex.GetTokenForSetQuote:
		if callbackParam.Exist {
			if err := dex.OnSetQuoteGetTokenInfoSuccess(db, callbackParam); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundGetTokenInfoCallback, err, sendBlock)
			}
		} else {
			if err := dex.OnSetQuoteGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundGetTokenInfoCallback, err, sendBlock)
			}
		}
	case dex.GetTokenForTransferOwner:
		if callbackParam.Exist {
			if err := dex.OnTransferOwnerGetTokenInfoSuccess(db, callbackParam); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundGetTokenInfoCallback, err, sendBlock)
			}
		} else {
			if err := dex.OnTransferOwnerGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundGetTokenInfoCallback, err, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfig, err, sendBlock)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigOwner) {
			dex.SetOwner(db, param.Owner)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTimer) {
			dex.SetTimerAddress(db, param.Timer)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTrigger) {
			dex.SetTriggerAddress(db, param.Trigger)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfig, dex.OnlyOwnerAllowErr, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, err, sendBlock)
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
						return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, dex.TradeMarketAllowMineErr, sendBlock)
					} else {
						return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, dex.TradeMarketNotAllowMineErr, sendBlock)
					}
				}
			} else {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, dex.TradeMarketNotExistsErr, sendBlock)
			}
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigNewQuoteToken) {
			if param.QuoteTokenType <= dex.ViteTokenType || param.QuoteTokenType > dex.UsdTokenType {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, dex.InvalidQuoteTokenTypeErr, sendBlock)
			}
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
					return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, dex.AlreadyQuoteType, sendBlock)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundOwnerConfigTrade, dex.OnlyOwnerAllowErr, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, err, sendBlock)
	}
	if marketInfo, ok = dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); !ok || !marketInfo.Valid {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, dex.TradeMarketNotExistsErr, sendBlock)
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
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, dex.InvalidBrokerFeeRateErr, sendBlock)
			}
			marketInfo.TakerBrokerFeeRate = param.TakerRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigMakerRate) {
			if !dex.ValidBrokerFeeRate(param.MakerRate) {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, dex.InvalidBrokerFeeRateErr, sendBlock)
			}
			marketInfo.MakerBrokerFeeRate = param.MakerRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerStopMarket) {
			marketInfo.Stopped = param.StopMarket
		}
		dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
		dex.AddMarketEvent(db, marketInfo)
	} else {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, dex.OnlyOwnerAllowErr, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundTransferTokenOwner, err, sendBlock)
	}
	if tokenInfo, ok := dex.GetTokenInfo(db, param.Token); ok {
		if bytes.Equal(tokenInfo.Owner, sendBlock.AccountAddress.Bytes()) {
			tokenInfo.Owner = param.Owner.Bytes()
			dex.SaveTokenInfo(db, param.Token, tokenInfo)
			dex.AddTokenEvent(db, tokenInfo)
		} else {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundTransferTokenOwner, dex.OnlyOwnerAllowErr, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNotifyTime, dex.InvalidSourceAddressErr, sendBlock)
	}
	if err = cabi.ABIDexFund.UnpackMethod(notifyTimeParam, cabi.MethodNameDexFundNotifyTime, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNotifyTime, err, sendBlock)
	}
	if err = dex.SetTimerTimestamp(db, notifyTimeParam.Timestamp, vm.ConsensusReader()); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNotifyTime, err, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewInviter, dex.AlreadyIsInviterErr, sendBlock)
	}
	if _, err := dex.SubUserFund(db, sendBlock.AccountAddress, ledger.ViteTokenId.Bytes(), dex.NewInviterFeeAmount); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewInviter, err, sendBlock)
	}
	dex.SettleFeesWithTokenId(db, vm.ConsensusReader(), true, ledger.ViteTokenId, dex.ViteTokenDecimals, dex.ViteTokenType, nil, dex.NewInviterFeeAmount, nil)
	if inviteCode := dex.NewInviteCode(db, block.PrevHash); inviteCode == 0 {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewInviter, dex.NewInviteCodeFailErr, sendBlock)
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
	return util.TotalGasCost(util.TxGas, data)
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundBindInviteCode, err, sendBlock)
	}
	if _, err = dex.GetInviterByInvitee(db, sendBlock.AccountAddress); err != dex.NotBindInviterErr {
		if err == nil {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundBindInviteCode, dex.AlreadyBindInviterErr, sendBlock)
		} else {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundBindInviteCode, err, sendBlock)
		}
	}
	if inviter, err = dex.GetInviterByCode(db, code); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundBindInviteCode, err, sendBlock)
	}
	dex.SaveInviterByInvitee(db, sendBlock.AccountAddress, *inviter)
	dex.AddInviteRelationEvent(db, *inviter, sendBlock.AccountAddress, code)
	return nil, nil
}

type MethodDexFundEndorseVxMinePool struct {
}

func (md *MethodDexFundEndorseVxMinePool) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundEndorseVxMinePool) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundEndorseVxMinePool) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(util.TxGas, data)
}

func (md *MethodDexFundEndorseVxMinePool) GetReceiveQuota() uint64 {
	return dexFundEndorseVxMinePoolReceiveGas
}

func (md *MethodDexFundEndorseVxMinePool) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 || block.TokenId != dex.VxTokenId {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundEndorseVxMinePool) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	poolAmount := dex.GetVxMinePool(db)
	poolAmount.Add(poolAmount, sendBlock.Amount)
	dex.SaveVxMinePool(db, poolAmount)
	return nil, nil
}

type MethodDexFundSettleMakerMinedVx struct {
}

func (md *MethodDexFundSettleMakerMinedVx) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleMakerMinedVx) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundSettleMakerMinedVx) GetSendQuota(data []byte) (uint64, error) {
	return util.TotalGasCost(util.TxGas, data)
}

func (md *MethodDexFundSettleMakerMinedVx) GetReceiveQuota() uint64 {
	return dexFundSettleMakerMinedVxReceiveGas
}

func (md *MethodDexFundSettleMakerMinedVx) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexSerializedData)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFunSettleMakerMinedVx, block.Data); err != nil {
		return err
	}
	actions := &dexproto.VxSettleActions{}
	if err = proto.Unmarshal(param.Data, actions); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundSettleMakerMinedVx) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err     error
		poolAmt *big.Int
		finish  bool
	)
	if !dex.IsMakerMineProxy(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFunSettleMakerMinedVx, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamDexSerializedData)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFunSettleMakerMinedVx, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFunSettleMakerMinedVx, err, sendBlock)
	}
	actions := &dexproto.VxSettleActions{}
	if err = proto.Unmarshal(param.Data, actions); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFunSettleMakerMinedVx, err, sendBlock)
	} else if len(actions.Actions) == 0 {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFunSettleMakerMinedVx, dex.InvalidInputParamErr, sendBlock)
	}
	if poolAmt = dex.GetMakerProxyAmountByPeriodId(db, actions.Period); poolAmt.Sign() == 0 {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFunSettleMakerMinedVx, dex.ExceedFundAvailableErr, sendBlock)
	}
	for _, action := range actions.Actions {
		if addr, err := types.BytesToAddress(action.Address); err != nil {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFunSettleMakerMinedVx, dex.InternalErr, sendBlock)
		} else {
			amt := new(big.Int).SetBytes(action.Amount)
			if amt.Cmp(poolAmt) > 0 {
				amt.Set(poolAmt)
			}
			acc := dex.DepositUserAccount(db, addr, dex.VxTokenId, amt)
			dex.OnDepositVx(db, vm.ConsensusReader(), addr, amt, acc)
			poolAmt.Sub(poolAmt, amt)
			if poolAmt.Sign() <= 0 {
				break
			}
		}
	}
	if poolAmt.Sign() > 0 {
		dex.SaveMakerProxyAmountByPeriodId(db, actions.Period, poolAmt)
	} else {
		dex.DeleteMakerProxyAmountByPeriodId(db, actions.Period)
		finish = true
	}
	dex.AddSettleMakerMinedVxEvent(db, actions.Period, actions.Page, finish)
	return nil, nil
}

func handleDexReceiveErr(logger log15.Logger, method string, err error, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	logger.Error("dex receive with err", "error", err.Error(), "method", method, "sendBlockHash", sendBlock.Hash.String(), "sendAddress", sendBlock.AccountAddress.String())
	return nil, err
}
