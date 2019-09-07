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

func (md *MethodDexFundUserDeposit) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundUserDeposit) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundDepositGas
}

func (md *MethodDexFundUserDeposit) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDexFundUserDeposit) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	account := dex.DepositUserAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount)
	// must do after account updated by deposit
	if sendBlock.TokenId == dex.VxTokenId {
		if err := dex.OnDepositVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundUserDeposit, err, sendBlock)
		}
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

func (md *MethodDexFundUserWithdraw) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundUserWithdraw) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundWithdrawGas
}

func (md *MethodDexFundUserWithdraw) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundWithDraw)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundUserWithdraw, block.Data); err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 {
		return dex.InvalidInputParamErr
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundUserWithdraw, err, sendBlock)
	} else {
		if param.Token == dex.VxTokenId {
			if err = dex.OnWithdrawVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, param.Amount, acc); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundUserWithdraw, err, sendBlock)
			}
		}
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: types.AddressDexFund,
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

func (md *MethodDexFundNewMarket) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundNewMarket) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewMarketGas
}

func (md *MethodDexFundNewMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundNewMarket)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewMarket, block.Data); err != nil {
		return err
	}
	if err = dex.CheckMarketParam(param); err != nil {
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
		if getTokenInfoData, err := dex.OnNewMarketPending(db, param, marketInfo); err != nil {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewMarket, err, sendBlock)
		} else {
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
}

type MethodDexFundNewOrder struct {
}

func (md *MethodDexFundNewOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewOrder) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewOrder) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundNewOrder) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewOrderGas
}

func (md *MethodDexFundNewOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	param := new(dex.ParamDexFundNewOrder)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewOrder, block.Data); err != nil {
		return err
	}
	return dex.PreCheckOrderParam(param, dex.IsStemFork(db))
}

func (md *MethodDexFundNewOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexFundNewOrder)
	cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewOrder, sendBlock.Data)
	if blocks, err := dex.DoNewOrder(db, param, &sendBlock.AccountAddress, nil, sendBlock.Hash); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewOrder, err, sendBlock)
	} else {
		return blocks, nil
	}
}

type MethodDexFundSettleOrders struct {
}

func (md *MethodDexFundSettleOrders) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleOrders) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundSettleOrders) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundSettleOrders) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundSettleOrdersGas
}

func (md *MethodDexFundSettleOrders) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	if block.AccountAddress != types.AddressDexTrade {
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
			if err = dex.DoSettleFund(db, vm.ConsensusReader(), fundAction, marketInfo, fundLogger); err != nil {
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

func (md *MethodDexFundPeriodJob) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPeriodJob) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPeriodJobGas
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
			vxPool, amount, vxPoolLeaved, refund *big.Int
			amtForItems                          map[int32]*big.Int
			success                              bool
		)
		vxPool = dex.GetVxMinePool(db)
		vxPoolLeaved = new(big.Int).Set(vxPool)
		switch param.BizType {
		case dex.MineVxForFeeJob:
			if amtForItems, vxPoolLeaved, success = dex.GetVxAmountsForEqualItems(db, param.PeriodId, vxPool, dex.RateSumForFeeMine, dex.ViteTokenType, dex.UsdTokenType); success {
				if refund, err = dex.DoMineVxForFee(db, vm.ConsensusReader(), param.PeriodId, amtForItems, fundLogger); err != nil {
					return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPeriodJob, fmt.Errorf("no vx available on mine for fee"), sendBlock)
			}
		case dex.MineVxForPledgeJob:
			if amount, vxPoolLeaved, success = dex.GetVxAmountToMine(db, param.PeriodId, vxPool, dex.RateForPledgeMine); success {
				if refund, err = dex.DoMineVxForPledge(db, vm.ConsensusReader(), param.PeriodId, amount); err != nil {
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
		if refund != nil && refund.Sign() > 0 {
			vxPoolLeaved.Add(vxPoolLeaved, refund)
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

func (md *MethodDexFundPledgeForVx) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPledgeForVx) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeForVxGas
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
	cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForVx, sendBlock.Data)
	if appendBlocks, err := dex.HandlePledgeAction(db, dex.PledgeForVx, param.ActionType, sendBlock.AccountAddress, param.Amount, nodeConfig.params.PledgeHeight); err != nil {
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

func (md *MethodDexFundPledgeForVip) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPledgeForVip) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeForVipGas
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
	cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForVip, sendBlock.Data)
	if appendBlocks, err := dex.HandlePledgeAction(db, dex.PledgeForVip, param.ActionType, sendBlock.AccountAddress, dex.PledgeForVipAmount, nodeConfig.params.ViteXVipPledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeForVip, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundPledgeForSuperVip struct {
}

func (md *MethodDexFundPledgeForSuperVip) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPledgeForSuperVip) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPledgeForSuperVip) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPledgeForSuperVip) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeForSuperVipGas
}

func (md *MethodDexFundPledgeForSuperVip) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamDexFundPledgeForVip)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForSuperVip, block.Data); err != nil {
		return err
	}
	if param.ActionType != dex.Pledge && param.ActionType != dex.CancelPledge {
		return dex.InvalidPledgeActionTypeErr
	}
	return nil
}

func (md MethodDexFundPledgeForSuperVip) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundPledgeForVip)
	cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundPledgeForSuperVip, sendBlock.Data)
	if appendBlocks, err := dex.HandlePledgeAction(db, dex.PledgeForSuperVip, param.ActionType, sendBlock.AccountAddress, dex.PledgeForSuperVipAmount, nodeConfig.params.ViteXSuperVipPledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeForSuperVip, err, sendBlock)
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

func (md *MethodDexFundPledgeCallback) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPledgeCallback) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeCallbackGas
}

func (md *MethodDexFundPledgeCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), cabi.MethodNameDexFundPledgeCallback, block.Data)
}

func (md MethodDexFundPledgeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var callbackParam = new(dex.ParamDexFundPledgeCallBack)
	cabi.ABIDexFund.UnpackMethod(callbackParam, cabi.MethodNameDexFundPledgeCallback, sendBlock.Data)
	if callbackParam.Success {
		switch callbackParam.Bid {
		case dex.PledgeForVx:
			pledgeAmount := dex.GetPledgeForVx(db, callbackParam.PledgeAddress)
			pledgeAmount.Add(pledgeAmount, callbackParam.Amount)
			dex.SavePledgeForVx(db, callbackParam.PledgeAddress, pledgeAmount)
			if err := dex.OnPledgeForVxSuccess(db, vm.ConsensusReader(), callbackParam.PledgeAddress, callbackParam.Amount, pledgeAmount); err != nil {
				handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundPledgeCallback, err, sendBlock)
			}
		case dex.PledgeForVip:
			if pledgeVip, ok := dex.GetPledgeForVip(db, callbackParam.PledgeAddress); ok { //duplicate pledge for vip
				pledgeVip.PledgeTimes = pledgeVip.PledgeTimes + 1
				dex.SavePledgeForVip(db, callbackParam.PledgeAddress, pledgeVip)
				// duplicate pledge for vip, cancel pledge
				return dex.DoCancelPledge(db, callbackParam.PledgeAddress, callbackParam.Bid, callbackParam.Amount)
			} else {
				pledgeVip.Timestamp = dex.GetTimestampInt64(db)
				pledgeVip.PledgeTimes = 1
				dex.SavePledgeForVip(db, callbackParam.PledgeAddress, pledgeVip)
			}
		case dex.PledgeForSuperVip:
			if pledgeSuperVip, ok := dex.GetPledgeForSuperVip(db, callbackParam.PledgeAddress); ok { //duplicate pledge for super vip
				pledgeSuperVip.PledgeTimes = pledgeSuperVip.PledgeTimes + 1
				dex.SavePledgeForSuperVip(db, callbackParam.PledgeAddress, pledgeSuperVip)
				// duplicate pledge for vip, cancel pledge
				return dex.DoCancelPledge(db, callbackParam.PledgeAddress, callbackParam.Bid, callbackParam.Amount)
			} else {
				pledgeSuperVip.Timestamp = dex.GetTimestampInt64(db)
				pledgeSuperVip.PledgeTimes = 1
				dex.SavePledgeForSuperVip(db, callbackParam.PledgeAddress, pledgeSuperVip)
			}
		}
	} else {
		switch callbackParam.Bid {
		case dex.PledgeForVx:
			if callbackParam.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
		case dex.PledgeForVip:
			if dex.PledgeForVipAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
		case dex.PledgeForSuperVip:
			if dex.PledgeForSuperVipAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
		}
		dex.DepositUserAccount(db, callbackParam.PledgeAddress, ledger.ViteTokenId, sendBlock.Amount)
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

func (md *MethodDexFundCancelPledgeCallback) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundCancelPledgeCallback) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundCancelPledgeCallbackGas
}

func (md *MethodDexFundCancelPledgeCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundPledgeCallBack), cabi.MethodNameDexFundCancelPledgeCallback, block.Data)
}

func (md MethodDexFundCancelPledgeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var cancelPledgeParam = new(dex.ParamDexFundPledgeCallBack)
	cabi.ABIDexFund.UnpackMethod(cancelPledgeParam, cabi.MethodNameDexFundCancelPledgeCallback, sendBlock.Data)
	if cancelPledgeParam.Success {
		switch cancelPledgeParam.Bid {
		case dex.PledgeForVx:
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
			if err := dex.OnCancelPledgeForVxSuccess(db, vm.ConsensusReader(), cancelPledgeParam.PledgeAddress, sendBlock.Amount, leaved); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundCancelPledgeCallback, err, sendBlock)
			}
		case dex.PledgeForVip:
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
		case dex.PledgeForSuperVip:
			if dex.PledgeForSuperVipAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForPledgeCallbackErr)
			}
			if pledgeSuperVip, ok := dex.GetPledgeForSuperVip(db, cancelPledgeParam.PledgeAddress); ok {
				pledgeSuperVip.PledgeTimes = pledgeSuperVip.PledgeTimes - 1
				if pledgeSuperVip.PledgeTimes == 0 {
					dex.DeletePledgeForSuperVip(db, cancelPledgeParam.PledgeAddress)
				} else {
					dex.SavePledgeForSuperVip(db, cancelPledgeParam.PledgeAddress, pledgeSuperVip)
				}
			} else {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundCancelPledgeCallback, dex.PledgeForSuperVipNotExistsErr, sendBlock)
			}
		}
		dex.DepositUserAccount(db, cancelPledgeParam.PledgeAddress, ledger.ViteTokenId, sendBlock.Amount)
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

func (md *MethodDexFundGetTokenInfoCallback) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundGetTokenInfoCallback) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundGetTokenInfoCallbackGas
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

func (md *MethodDexFundOwnerConfig) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundOwnerConfig) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundOwnerConfigGas
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

func (md *MethodDexFundOwnerConfigTrade) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundOwnerConfigTrade) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundOwnerConfigTradeGas
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
			if param.QuoteTokenType < dex.ViteTokenType || param.QuoteTokenType > dex.UsdTokenType {
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
		if dex.IsStemFork(db) && dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigStartNormalMine) {
			dex.StartNormalMine(db)
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

func (md *MethodDexFundMarketOwnerConfig) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundMarketOwnerConfig) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundMarketOwnerConfigGas
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
			if !dex.ValidBrokerFeeRate(param.TakerFeeRate) {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, dex.InvalidBrokerFeeRateErr, sendBlock)
			}
			marketInfo.TakerBrokerFeeRate = param.TakerFeeRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigMakerRate) {
			if !dex.ValidBrokerFeeRate(param.MakerFeeRate) {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundMarketOwnerConfig, dex.InvalidBrokerFeeRateErr, sendBlock)
			}
			marketInfo.MakerBrokerFeeRate = param.MakerFeeRate
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

func (md *MethodDexFundTransferTokenOwner) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundTransferTokenOwner) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundTransferTokenOwnerGas
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

func (md *MethodDexFundNotifyTime) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundNotifyTime) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNotifyTimeGas
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

func (md *MethodDexFundNewInviter) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundNewInviter) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewInviterGas
}

func (md *MethodDexFundNewInviter) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
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

func (md *MethodDexFundBindInviteCode) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundBindInviteCode) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundBindInviteCodeGas
}

func (md *MethodDexFundBindInviteCode) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var code uint32
	if err := cabi.ABIDexFund.UnpackMethod(&code, cabi.MethodNameDexFundBindInviteCode, block.Data); err != nil {
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

func (md *MethodDexFundEndorseVxMinePool) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundEndorseVxMinePool) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundEndorseVxMinePoolGas
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

func (md *MethodDexFundSettleMakerMinedVx) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundSettleMakerMinedVx) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundSettleMakerMinedVxGas
}

func (md *MethodDexFundSettleMakerMinedVx) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexSerializedData)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundSettleMakerMinedVx, block.Data); err != nil {
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
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamDexSerializedData)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundSettleMakerMinedVx, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, err, sendBlock)
	}
	actions := &dexproto.VxSettleActions{}
	if err = proto.Unmarshal(param.Data, actions); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, err, sendBlock)
	} else if len(actions.Actions) == 0 {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, dex.InvalidInputParamErr, sendBlock)
	} else {
		if lastPeriod := dex.GetLastSettledMakerMinedVxPeriod(db); lastPeriod > 0 && actions.Period != lastPeriod+1 {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, dex.InvalidInputParamErr, sendBlock)
		}
		if lastPageId := dex.GetLastSettledMakerMinedVxPage(db); lastPageId > 0 && actions.Page != lastPageId+1 {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, dex.InvalidInputParamErr, sendBlock)
		}
	}
	if poolAmt = dex.GetMakerProxyAmountByPeriodId(db, actions.Period); poolAmt.Sign() == 0 {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, dex.ExceedFundAvailableErr, sendBlock)
	}
	for _, action := range actions.Actions {
		if addr, err := types.BytesToAddress(action.Address); err != nil {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, dex.InternalErr, sendBlock)
		} else {
			amt := new(big.Int).SetBytes(action.Amount)
			if amt.Cmp(poolAmt) > 0 {
				amt.Set(poolAmt)
			}
			acc := dex.DepositUserAccount(db, addr, dex.VxTokenId, amt)
			if err = dex.OnDepositVx(db, vm.ConsensusReader(), addr, amt, acc); err != nil {
				return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundSettleMakerMinedVx, err, sendBlock)
			}
			poolAmt.Sub(poolAmt, amt)
			if poolAmt.Sign() <= 0 {
				break
			}
		}
	}
	if poolAmt.Sign() > 0 {
		dex.SaveMakerProxyAmountByPeriodId(db, actions.Period, poolAmt)
		dex.SaveLastSettledMakerMinedVxPage(db, actions.Page)
	} else {
		finish = true
		dex.DeleteMakerProxyAmountByPeriodId(db, actions.Period)
		dex.SaveLastSettledMakerMinedVxPeriod(db, actions.Period)
		dex.DeleteLastSettledMakerMinedVxPage(db)
	}
	dex.AddSettleMakerMinedVxEvent(db, actions.Period, actions.Page, finish)
	return nil, nil
}

type MethodDexFundConfigMarketsAgent struct {
}

func (md *MethodDexFundConfigMarketsAgent) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundConfigMarketsAgent) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundConfigMarketsAgent) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundConfigMarketsAgent) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundConfigMarketsAgentGas
}

func (md *MethodDexFundConfigMarketsAgent) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var param = new(dex.ParamDexFundConfigMarketsAgent)
	if err := cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundConfigMarketsAgent, block.Data); err != nil {
		return err
	} else if param.ActionType != dex.GrantAgent && param.ActionType != dex.RevokeAgent || len(param.TradeTokens) == 0 || len(param.TradeTokens) != len(param.QuoteTokens) || block.AccountAddress == param.Agent {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundConfigMarketsAgent) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamDexFundConfigMarketsAgent)
		err   error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundConfigMarketsAgent, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundConfigMarketsAgent, err, sendBlock)
	}
	for i, tradeToken := range param.TradeTokens {
		if marketInfo, ok := dex.GetMarketInfo(db, tradeToken, param.QuoteTokens[i]); !ok {
			return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundConfigMarketsAgent, dex.TradeMarketNotExistsErr, sendBlock)
		} else {
			switch param.ActionType {
			case dex.GrantAgent:
				if !dex.IsMarketGrantedToAgent(db, sendBlock.AccountAddress, param.Agent, marketInfo.MarketId) {
					dex.GrantMarketToAgent(db, sendBlock.AccountAddress, param.Agent, marketInfo.MarketId)
					dex.AddGrantMarketToAgentEvent(db, sendBlock.AccountAddress, param.Agent, marketInfo.MarketId)
				}
			case dex.RevokeAgent:
				if dex.IsMarketGrantedToAgent(db, sendBlock.AccountAddress, param.Agent, marketInfo.MarketId) {
					dex.RevokeMarketFromAgent(db, sendBlock.AccountAddress, param.Agent, marketInfo.MarketId)
					dex.AddRevokeMarketFromAgentEvent(db, sendBlock.AccountAddress, param.Agent, marketInfo.MarketId)
				}
			}
		}
	}
	return nil, nil
}

type MethodDexFundNewAgentOrder struct {
}

func (md *MethodDexFundNewAgentOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNewAgentOrder) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNewAgentOrder) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundNewAgentOrder) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewAgentOrderGas
}

func (md *MethodDexFundNewAgentOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(dex.ParamDexFundNewAgentOrder)
	if err := cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewAgentOrder, block.Data); err != nil {
		return err
	}
	return dex.PreCheckOrderParam(&param.ParamDexFundNewOrder, true)
}

func (md MethodDexFundNewAgentOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamDexFundNewAgentOrder)
		err   error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, cabi.MethodNameDexFundNewAgentOrder, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewAgentOrder, err, sendBlock)
	}
	if blocks, err := dex.DoNewOrder(db, &param.ParamDexFundNewOrder, &param.Principal, &sendBlock.AccountAddress, sendBlock.Hash); err != nil {
		return handleDexReceiveErr(fundLogger, cabi.MethodNameDexFundNewAgentOrder, err, sendBlock)
	} else {
		return blocks, nil
	}
}

func handleDexReceiveErr(logger log15.Logger, method string, err error, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	logger.Error("dex receive with err", "error", err.Error(), "method", method, "sendBlockHash", sendBlock.Hash.String(), "sendAddress", sendBlock.AccountAddress.String())
	return nil, err
}
