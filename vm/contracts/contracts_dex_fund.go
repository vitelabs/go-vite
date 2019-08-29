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

var method BuiltinContractMethod = &MethodDexFundDeposit{}

type MethodDexFundDeposit struct {
	MethodName string
}

func (md *MethodDexFundDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundDeposit) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundDeposit) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundDeposit) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundDepositGas
}

func (md *MethodDexFundDeposit) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDexFundDeposit) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	account := dex.DepositAccount(db, sendBlock.AccountAddress, sendBlock.TokenId, sendBlock.Amount)
	// must do after account updated by deposit
	if sendBlock.TokenId == dex.VxTokenId {
		if err := dex.OnDepositVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, sendBlock.Amount, account); err != nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
		}
	}
	return nil, nil
}

type MethodDexFundWithdraw struct {
	MethodName string
}

func (md *MethodDexFundWithdraw) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundWithdraw) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundWithdraw) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundWithdraw) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundWithdrawGas
}

func (md *MethodDexFundWithdraw) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundWithDraw)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDexFundWithdraw) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexFundWithDraw)
	var (
		acc *dexproto.Account
		err error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if acc, err = dex.ReduceAccount(db, sendBlock.AccountAddress, param.Token.Bytes(), param.Amount); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		if param.Token == dex.VxTokenId {
			if err = dex.OnWithdrawVx(db, vm.ConsensusReader(), sendBlock.AccountAddress, param.Amount, acc); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
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

type MethodDexFundOpenNewMarket struct {
	MethodName string
}

func (md *MethodDexFundOpenNewMarket) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundOpenNewMarket) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundOpenNewMarket) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundOpenNewMarket) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewMarketGas
}

func (md *MethodDexFundOpenNewMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamDexFundNewMarket)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	if err = dex.CheckMarketParam(param); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundOpenNewMarket) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	param := new(dex.ParamDexFundNewMarket)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return nil, err
	}
	if mk, ok := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); ok && mk.Valid { // if mk not valid, overwrite old marketInfo with new
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketExistsErr, sendBlock)
	}
	marketInfo := &dex.MarketInfo{}
	if err = dex.RenderMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken, nil, &sendBlock.AccountAddress); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if marketInfo.Valid {
		if appendBlocks, err := dex.OnNewMarketValid(db, vm.ConsensusReader(), marketInfo, param.TradeToken, param.QuoteToken, &sendBlock.AccountAddress); err == nil {
			return appendBlocks, nil
		} else {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
		}
	} else {
		if getTokenInfoData, err := dex.OnNewMarketPending(db, param, marketInfo); err != nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
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

type MethodDexFundPlaceOrder struct {
	MethodName string
}

func (md *MethodDexFundPlaceOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPlaceOrder) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPlaceOrder) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPlaceOrder) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewOrderGas
}

func (md *MethodDexFundPlaceOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	param := new(dex.ParamDexFundNewOrder)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	return dex.PreCheckOrderParam(param, dex.IsStemFork(db))
}

func (md *MethodDexFundPlaceOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamDexFundNewOrder)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if blocks, err := dex.DoNewOrder(db, param, &sendBlock.AccountAddress, nil, sendBlock.Hash); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return blocks, nil
	}
}

type MethodDexFundSettleOrders struct {
	MethodName string
}

func (md *MethodDexFundSettleOrders) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleOrders) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	settleActions := &dexproto.SettleActions{}
	if err = proto.Unmarshal(param.Data, settleActions); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if marketInfo, ok := dex.GetMarketInfoByTokens(db, settleActions.TradeToken, settleActions.QuoteToken); !ok {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketNotExistsErr, sendBlock)
	} else {
		for _, fundAction := range settleActions.FundActions {
			if err = dex.DoSettleFund(db, vm.ConsensusReader(), fundAction, marketInfo, fundLogger); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		}
		if len(settleActions.FeeActions) > 0 {
			dex.SettleFees(db, vm.ConsensusReader(), marketInfo.AllowMining, marketInfo.QuoteToken, marketInfo.QuoteTokenDecimals, marketInfo.QuoteTokenType, settleActions.FeeActions, nil, nil)
			dex.SettleBrokerFeeSum(db, vm.ConsensusReader(), settleActions.FeeActions, marketInfo)
		}
		return nil, nil
	}
}

type MethodDexFundTriggerPeriodJob struct {
	MethodName string
}

func (md *MethodDexFundTriggerPeriodJob) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundTriggerPeriodJob) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundTriggerPeriodJob) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundTriggerPeriodJob) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPeriodJobGas
}

func (md *MethodDexFundTriggerPeriodJob) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexPeriodJob), md.MethodName, block.Data)
}

func (md MethodDexFundTriggerPeriodJob) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err error
	)
	if !dex.ValidTriggerAddress(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamDexPeriodJob)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if param.PeriodId >= dex.GetCurrentPeriodId(db, vm.ConsensusReader()) {
		return handleDexReceiveErr(fundLogger, md.MethodName, fmt.Errorf("job periodId for biz %d not before current periodId", param.BizType), sendBlock)
	}
	if lastPeriodId := dex.GetLastJobPeriodIdByBizType(db, param.BizType); lastPeriodId > 0 && param.PeriodId != lastPeriodId+1 {
		return handleDexReceiveErr(fundLogger, md.MethodName, fmt.Errorf("job periodId for biz %d  not equals to expected id %d", param.BizType, lastPeriodId+1), sendBlock)
	}
	dex.AddPeriodWithBizEvent(db, param.PeriodId, param.BizType)
	if param.BizType <= dex.OperatorFeeDividendJob {
		switch param.BizType {
		case dex.FeeDividendJob:
			err = dex.DoDivideFees(db, param.PeriodId)
		case dex.OperatorFeeDividendJob:
			err = dex.DoDivideOperatorFees(db, param.PeriodId)
		}
		if err != nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
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
					return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, fmt.Errorf("no vx available on mine for fee"), sendBlock)
			}
		case dex.MineVxForStakingJob:
			if amount, vxPoolLeaved, success = dex.GetVxAmountToMine(db, param.PeriodId, vxPool, dex.RateForStakingMine); success {
				if refund, err = dex.DoMineVxForStaking(db, vm.ConsensusReader(), param.PeriodId, amount); err != nil {
					return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, fmt.Errorf("no vx available on mine for pledge"), sendBlock)
			}
		case dex.MineVxForMakerAndMaintainerJob:
			if amtForItems, vxPoolLeaved, success = dex.GetVxAmountsForEqualItems(db, param.PeriodId, vxPool, dex.RateSumForMakerAndMaintainerMine, dex.MineForMaker, dex.MineForMaintainer); success {
				if err = dex.DoMineVxForMakerMineAndMaintainer(db, param.PeriodId, vm.ConsensusReader(), amtForItems); err != nil {
					return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, fmt.Errorf("no vx available on mine for maker and maintainer"), sendBlock)
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

type MethodDexFundStakeForMining struct {
	MethodName string
}

func (md *MethodDexFundStakeForMining) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundStakeForMining) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundStakeForMining) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundStakeForMining) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeForVxGas
}

func (md *MethodDexFundStakeForMining) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamDexFundStakeForMining)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else {
		if param.Amount.Cmp(dex.StakeForVxMinAmount) < 0 {
			return dex.InvalidStakeAmountErr
		}
		if param.ActionType != dex.Stake && param.ActionType != dex.CancelStake {
			return dex.InvalidStakeActionTypeErr
		}
	}
	return nil
}

func (md MethodDexFundStakeForMining) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundStakeForMining)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForVx, param.ActionType, sendBlock.AccountAddress, param.Amount, nodeConfig.params.PledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundStakeForVIP struct {
	MethodName string
}

func (md *MethodDexFundStakeForVIP) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundStakeForVIP) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundStakeForVIP) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundStakeForVIP) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeForVipGas
}

func (md *MethodDexFundStakeForVIP) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamDexFundStakeForVIP)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	if param.ActionType != dex.Stake && param.ActionType != dex.CancelStake {
		return dex.InvalidStakeActionTypeErr
	}
	return nil
}

func (md MethodDexFundStakeForVIP) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundStakeForVIP)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForVIP, param.ActionType, sendBlock.AccountAddress, dex.StakeForVIPAmount, nodeConfig.params.ViteXVipPledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundStakeForSVIP struct {
	MethodName string
}

func (md *MethodDexFundStakeForSVIP) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundStakeForSVIP) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundStakeForSVIP) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundStakeForSVIP) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeForSuperVipGas
}

func (md *MethodDexFundStakeForSVIP) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamDexFundStakeForVIP)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	if param.ActionType != dex.Stake && param.ActionType != dex.CancelStake {
		return dex.InvalidStakeActionTypeErr
	}
	return nil
}

func (md MethodDexFundStakeForSVIP) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDexFundStakeForVIP)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForSuperVIP, param.ActionType, sendBlock.AccountAddress, dex.StakeForSuperVIPAmount, nodeConfig.params.ViteXSuperVipPledgeHeight); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundDelegateStakingCallback struct {
	MethodName string
}

func (md *MethodDexFundDelegateStakingCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundDelegateStakingCallback) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundDelegateStakingCallback) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundDelegateStakingCallback) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundPledgeCallbackGas
}

func (md *MethodDexFundDelegateStakingCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundStakeCallBack), md.MethodName, block.Data)
}

func (md MethodDexFundDelegateStakingCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var callbackParam = new(dex.ParamDexFundStakeCallBack)
	cabi.ABIDexFund.UnpackMethod(callbackParam, md.MethodName, sendBlock.Data)
	if callbackParam.Success {
		switch callbackParam.Bid {
		case dex.StakeForVx:
			pledgeAmount := dex.GetMiningStakedAmount(db, callbackParam.StakeAddr)
			pledgeAmount.Add(pledgeAmount, callbackParam.Amount)
			dex.SaveMiningStakedAmount(db, callbackParam.StakeAddr, pledgeAmount)
			if err := dex.OnMiningStakeSuccess(db, vm.ConsensusReader(), callbackParam.StakeAddr, callbackParam.Amount, pledgeAmount); err != nil {
				handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		case dex.StakeForVIP:
			if vipStaking, ok := dex.GetVIPStaking(db, callbackParam.StakeAddr); ok { //duplicate pledge for vip
				vipStaking.StakedTimes = vipStaking.StakedTimes + 1
				dex.SaveVIPStaking(db, callbackParam.StakeAddr, vipStaking)
				// duplicate pledge for vip, cancel pledge
				return dex.DoCancelStake(db, callbackParam.StakeAddr, callbackParam.Bid, callbackParam.Amount)
			} else {
				vipStaking.Timestamp = dex.GetTimestampInt64(db)
				vipStaking.StakedTimes = 1
				dex.SaveVIPStaking(db, callbackParam.StakeAddr, vipStaking)
			}
		case dex.StakeForSuperVIP:
			if superVIPStaing, ok := dex.GetSuperVIPStaking(db, callbackParam.StakeAddr); ok { //duplicate pledge for super vip
				superVIPStaing.StakedTimes = superVIPStaing.StakedTimes + 1
				dex.SaveSuperVIPStaking(db, callbackParam.StakeAddr, superVIPStaing)
				// duplicate pledge for vip, cancel pledge
				return dex.DoCancelStake(db, callbackParam.StakeAddr, callbackParam.Bid, callbackParam.Amount)
			} else {
				superVIPStaing.Timestamp = dex.GetTimestampInt64(db)
				superVIPStaing.StakedTimes = 1
				dex.SaveSuperVIPStaking(db, callbackParam.StakeAddr, superVIPStaing)
			}
		}
	} else {
		switch callbackParam.Bid {
		case dex.StakeForVx:
			if callbackParam.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
		case dex.StakeForVIP:
			if dex.StakeForVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
		case dex.StakeForSuperVIP:
			if dex.StakeForSuperVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
		}
		dex.DepositAccount(db, callbackParam.StakeAddr, ledger.ViteTokenId, sendBlock.Amount)
	}
	return nil, nil
}

type MethodDexFundCancelDelegatedStakingCallback struct {
	MethodName string
}

func (md *MethodDexFundCancelDelegatedStakingCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelDelegatedStakingCallback) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundCancelDelegatedStakingCallback) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundCancelDelegatedStakingCallback) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundCancelPledgeCallbackGas
}

func (md *MethodDexFundCancelDelegatedStakingCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressPledge {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundStakeCallBack), md.MethodName, block.Data)
}

func (md MethodDexFundCancelDelegatedStakingCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var cancelPledgeParam = new(dex.ParamDexFundStakeCallBack)
	cabi.ABIDexFund.UnpackMethod(cancelPledgeParam, md.MethodName, sendBlock.Data)
	if cancelPledgeParam.Success {
		switch cancelPledgeParam.Bid {
		case dex.StakeForVx:
			if cancelPledgeParam.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			pledgeAmount := dex.GetMiningStakedAmount(db, cancelPledgeParam.StakeAddr)
			leaved := new(big.Int).Sub(pledgeAmount, sendBlock.Amount)
			if leaved.Sign() < 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			} else if leaved.Sign() == 0 {
				dex.DeleteMiningStakedAmount(db, cancelPledgeParam.StakeAddr)
			} else {
				dex.SaveMiningStakedAmount(db, cancelPledgeParam.StakeAddr, leaved)
			}
			if err := dex.OnCancelMiningStakeSuccess(db, vm.ConsensusReader(), cancelPledgeParam.StakeAddr, sendBlock.Amount, leaved); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		case dex.StakeForVIP:
			if dex.StakeForVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			if vipStaking, ok := dex.GetVIPStaking(db, cancelPledgeParam.StakeAddr); ok {
				vipStaking.StakedTimes = vipStaking.StakedTimes - 1
				if vipStaking.StakedTimes == 0 {
					dex.DeleteVIPStaking(db, cancelPledgeParam.StakeAddr)
				} else {
					dex.SaveVIPStaking(db, cancelPledgeParam.StakeAddr, vipStaking)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.VIPStakingNotExistsErr, sendBlock)
			}
		case dex.StakeForSuperVIP:
			if dex.StakeForSuperVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			if superVIPStaing, ok := dex.GetSuperVIPStaking(db, cancelPledgeParam.StakeAddr); ok {
				superVIPStaing.StakedTimes = superVIPStaing.StakedTimes - 1
				if superVIPStaing.StakedTimes == 0 {
					dex.DeleteSuperVIPStaking(db, cancelPledgeParam.StakeAddr)
				} else {
					dex.SaveSuperVIPStaking(db, cancelPledgeParam.StakeAddr, superVIPStaing)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.SuperVIPStakingNotExistsErr, sendBlock)
			}
		}
		dex.DepositAccount(db, cancelPledgeParam.StakeAddr, ledger.ViteTokenId, sendBlock.Amount)
	}
	return nil, nil
}

type MethodDexFundGetTokenInfoCallback struct {
	MethodName string
}

func (md *MethodDexFundGetTokenInfoCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundGetTokenInfoCallback) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
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
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundGetTokenInfoCallback), md.MethodName, block.Data)
}

func (md MethodDexFundGetTokenInfoCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var callbackParam = new(dex.ParamDexFundGetTokenInfoCallback)
	cabi.ABIDexFund.UnpackMethod(callbackParam, md.MethodName, sendBlock.Data)
	switch callbackParam.Bid {
	case dex.GetTokenForNewMarket:
		if callbackParam.Exist {
			if appendBlocks, err := dex.OnNewMarketGetTokenInfoSuccess(db, vm.ConsensusReader(), callbackParam.TokenId, callbackParam); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			} else {
				return appendBlocks, nil
			}
		} else {
			if err := dex.OnNewMarketGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		}
	case dex.GetTokenForSetQuote:
		if callbackParam.Exist {
			if err := dex.OnSetQuoteGetTokenInfoSuccess(db, callbackParam); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		} else {
			if err := dex.OnSetQuoteGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		}
	case dex.GetTokenForTransferOwner:
		if callbackParam.Exist {
			if err := dex.OnTransferOwnerGetTokenInfoSuccess(db, callbackParam); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		} else {
			if err := dex.OnTransferOwnerGetTokenInfoFailed(db, callbackParam.TokenId); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		}
	}
	return nil, nil
}

type MethodDexFundDexAdminConfig struct {
	MethodName string
}

func (md *MethodDexFundDexAdminConfig) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundDexAdminConfig) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundDexAdminConfig) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundDexAdminConfig) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundOwnerConfigGas
}

func (md *MethodDexFundDexAdminConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundDexAdminConfig), md.MethodName, block.Data)
}

func (md MethodDexFundDexAdminConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexFundDexAdminConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigOwner) {
			dex.SetOwner(db, param.Owner)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTimer) {
			dex.SetTimerAddress(db, param.TimeOracle)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTrigger) {
			dex.SetTriggerAddress(db, param.PeriodJobTrigger)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigStopViteX) {
			dex.SaveViteXStopped(db, param.StopDex)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMakerMineProxy) {
			dex.SaveMakerMineProxy(db, param.MakerMiningAdmin)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMaintainer) {
			dex.SaveMaintainer(db, param.Maintainer)
		}
	} else {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.OnlyOwnerAllowErr, sendBlock)
	}
	return nil, nil
}

type MethodDexFundTradeAdminConfig struct {
	MethodName string
}

func (md *MethodDexFundTradeAdminConfig) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundTradeAdminConfig) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundTradeAdminConfig) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundTradeAdminConfig) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundOwnerConfigTradeGas
}

func (md *MethodDexFundTradeAdminConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundTradeAdminConfig), md.MethodName, block.Data)
}

func (md MethodDexFundTradeAdminConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexFundTradeAdminConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMineMarket) {
			if marketInfo, ok := dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); ok && marketInfo.Valid {
				if param.AllowMining != marketInfo.AllowMining {
					marketInfo.AllowMining = param.AllowMining
					dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
					dex.AddMarketEvent(db, marketInfo)
				} else {
					if marketInfo.AllowMining {
						return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketAllowMineErr, sendBlock)
					} else {
						return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketNotAllowMineErr, sendBlock)
					}
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketNotExistsErr, sendBlock)
			}
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigNewQuoteToken) {
			if param.QuoteTokenType < dex.ViteTokenType || param.QuoteTokenType > dex.UsdTokenType {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidQuoteTokenTypeErr, sendBlock)
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
					return handleDexReceiveErr(fundLogger, md.MethodName, dex.AlreadyQuoteType, sendBlock)
				} else {
					tokenInfo.QuoteTokenType = int32(param.QuoteTokenType)
					dex.SaveTokenInfo(db, param.NewQuoteToken, tokenInfo)
					dex.AddTokenEvent(db, tokenInfo)
				}
			}
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigTradeThreshold) {
			dex.SaveTradeThreshold(db, param.TokenTypeForTradeThreshold, param.MinTradeThreshold)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigMineThreshold) {
			dex.SaveMineThreshold(db, param.TokenTypeForMiningThreshold, param.MinMiningThreshold)
		}
		if dex.IsStemFork(db) && dex.IsOperationValidWithMask(param.OperationCode, dex.OwnerConfigStartNormalMine) {
			dex.StartNormalMine(db)
		}
	} else {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.OnlyOwnerAllowErr, sendBlock)
	}
	return nil, nil
}

type MethodDexFundMarketAdminConfig struct {
	MethodName string
}

func (md *MethodDexFundMarketAdminConfig) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundMarketAdminConfig) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundMarketAdminConfig) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundMarketAdminConfig) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundMarketOwnerConfigGas
}

func (md *MethodDexFundMarketAdminConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundMarketOwnerConfig), md.MethodName, block.Data)
}

func (md MethodDexFundMarketAdminConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err        error
		marketInfo *dex.MarketInfo
		ok         bool
	)
	var param = new(dex.ParamDexFundMarketOwnerConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if marketInfo, ok = dex.GetMarketInfo(db, param.TradeToken, param.QuoteToken); !ok || !marketInfo.Valid {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketNotExistsErr, sendBlock)
	}
	if bytes.Equal(sendBlock.AccountAddress.Bytes(), marketInfo.Owner) {
		if param.OperationCode == 0 {
			return nil, nil
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerTransferOwner) {
			marketInfo.Owner = param.MarketOwner.Bytes()
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigTakerRate) {
			if !dex.ValidBrokerFeeRate(param.TakerFeeRate) {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidBrokerFeeRateErr, sendBlock)
			}
			marketInfo.TakerOperatorFeeRate = param.TakerFeeRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigMakerRate) {
			if !dex.ValidBrokerFeeRate(param.MakerFeeRate) {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidBrokerFeeRateErr, sendBlock)
			}
			marketInfo.MakerOperatorFeeRate = param.MakerFeeRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerStopMarket) {
			marketInfo.Stopped = param.StopMarket
		}
		dex.SaveMarketInfo(db, marketInfo, param.TradeToken, param.QuoteToken)
		dex.AddMarketEvent(db, marketInfo)
	} else {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.OnlyOwnerAllowErr, sendBlock)
	}
	return nil, nil
}

type MethodDexFundTransferTokenOwnership struct {
	MethodName string
}

func (md *MethodDexFundTransferTokenOwnership) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundTransferTokenOwnership) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundTransferTokenOwnership) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundTransferTokenOwnership) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundTransferTokenOwnerGas
}

func (md *MethodDexFundTransferTokenOwnership) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundTransferTokenOwnership), md.MethodName, block.Data)
}

func (md MethodDexFundTransferTokenOwnership) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err   error
		param = new(dex.ParamDexFundTransferTokenOwnership)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if tokenInfo, ok := dex.GetTokenInfo(db, param.Token); ok {
		if bytes.Equal(tokenInfo.Owner, sendBlock.AccountAddress.Bytes()) {
			tokenInfo.Owner = param.NewOwner.Bytes()
			dex.SaveTokenInfo(db, param.Token, tokenInfo)
			dex.AddTokenEvent(db, tokenInfo)
		} else {
			return handleDexReceiveErr(fundLogger, md.MethodName, dex.OnlyOwnerAllowErr, sendBlock)
		}
	} else {
		getTokenInfoData := dex.OnTransferTokenOwnerPending(db, param.Token, sendBlock.AccountAddress, param.NewOwner)
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
	MethodName string
}

func (md *MethodDexFundNotifyTime) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundNotifyTime) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundNotifyTime) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundNotifyTime) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNotifyTimeGas
}

func (md *MethodDexFundNotifyTime) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexFundNotifyTime), md.MethodName, block.Data)
}

func (md MethodDexFundNotifyTime) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		notifyTimeParam = new(dex.ParamDexFundNotifyTime)
	)
	if !dex.ValidTimerAddress(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidSourceAddressErr, sendBlock)
	}
	if err = cabi.ABIDexFund.UnpackMethod(notifyTimeParam, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if err = dex.SetTimerTimestamp(db, notifyTimeParam.Timestamp, vm.ConsensusReader()); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	return nil, nil
}

type MethodDexFundCreateNewInviter struct {
	MethodName string
}

func (md *MethodDexFundCreateNewInviter) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCreateNewInviter) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundCreateNewInviter) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundCreateNewInviter) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewInviterGas
}

func (md *MethodDexFundCreateNewInviter) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return nil
}

func (md MethodDexFundCreateNewInviter) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	if code := dex.GetCodeByInviter(db, sendBlock.AccountAddress); code > 0 {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.AlreadyIsInviterErr, sendBlock)
	}
	if _, err := dex.ReduceAccount(db, sendBlock.AccountAddress, ledger.ViteTokenId.Bytes(), dex.NewInviterFeeAmount); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	dex.SettleFeesWithTokenId(db, vm.ConsensusReader(), true, ledger.ViteTokenId, dex.ViteTokenDecimals, dex.ViteTokenType, nil, dex.NewInviterFeeAmount, nil)
	if inviteCode := dex.NewInviteCode(db, block.PrevHash); inviteCode == 0 {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.NewInviteCodeFailErr, sendBlock)
	} else {
		dex.SaveCodeByInviter(db, sendBlock.AccountAddress, inviteCode)
		dex.SaveInviterByCode(db, sendBlock.AccountAddress, inviteCode)
	}
	return nil, nil
}

type MethodDexFundBindInviteCode struct {
	MethodName string
}

func (md *MethodDexFundBindInviteCode) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundBindInviteCode) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
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
	if err := cabi.ABIDexFund.UnpackMethod(&code, md.MethodName, block.Data); err != nil {
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
	if err = cabi.ABIDexFund.UnpackMethod(&code, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if _, err = dex.GetInviterByInvitee(db, sendBlock.AccountAddress); err != dex.NotBindInviterErr {
		if err == nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, dex.AlreadyBindInviterErr, sendBlock)
		} else {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
		}
	}
	if inviter, err = dex.GetInviterByCode(db, code); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	dex.SaveInviterByInvitee(db, sendBlock.AccountAddress, *inviter)
	dex.AddInviteRelationEvent(db, *inviter, sendBlock.AccountAddress, code)
	return nil, nil
}

type MethodDexFundEndorseVx struct {
	MethodName string
}

func (md *MethodDexFundEndorseVx) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundEndorseVx) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundEndorseVx) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundEndorseVx) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundEndorseVxMinePoolGas
}

func (md *MethodDexFundEndorseVx) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() <= 0 || block.TokenId != dex.VxTokenId {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundEndorseVx) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	poolAmount := dex.GetVxMinePool(db)
	poolAmount.Add(poolAmount, sendBlock.Amount)
	dex.SaveVxMinePool(db, poolAmount)
	return nil, nil
}

type MethodDexFundSettleMakerMinedVx struct {
	MethodName string
}

func (md *MethodDexFundSettleMakerMinedVx) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSettleMakerMinedVx) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
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
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
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
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamDexSerializedData)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	actions := &dexproto.VxSettleActions{}
	if err = proto.Unmarshal(param.Data, actions); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else if len(actions.Actions) == 0 {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidInputParamErr, sendBlock)
	} else {
		if lastPeriod := dex.GetLastSettledMakerMinedVxPeriod(db); lastPeriod > 0 && actions.Period != lastPeriod+1 {
			return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidInputParamErr, sendBlock)
		}
		if lastPageId := dex.GetLastSettledMakerMinedVxPage(db); lastPageId > 0 && actions.Page != lastPageId+1 {
			return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidInputParamErr, sendBlock)
		}
	}
	if poolAmt = dex.GetMakerProxyAmountByPeriodId(db, actions.Period); poolAmt.Sign() == 0 {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.ExceedFundAvailableErr, sendBlock)
	}
	for _, action := range actions.Actions {
		if addr, err := types.BytesToAddress(action.Address); err != nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, dex.InternalErr, sendBlock)
		} else {
			amt := new(big.Int).SetBytes(action.Amount)
			if amt.Cmp(poolAmt) > 0 {
				amt.Set(poolAmt)
			}
			acc := dex.DepositAccount(db, addr, dex.VxTokenId, amt)
			if err = dex.OnDepositVx(db, vm.ConsensusReader(), addr, amt, acc); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
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

type MethodDexFundConfigMarketAgents struct {
	MethodName string
}

func (md *MethodDexFundConfigMarketAgents) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundConfigMarketAgents) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundConfigMarketAgents) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundConfigMarketAgents) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundConfigMarketsAgentGas
}

func (md *MethodDexFundConfigMarketAgents) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var param = new(dex.ParamDexFundConfigMarketAgents)
	if err := cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else if param.ActionType != dex.GrantAgent && param.ActionType != dex.RevokeAgent || len(param.TradeTokens) == 0 || len(param.TradeTokens) != len(param.QuoteTokens) || block.AccountAddress == param.Agent {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundConfigMarketAgents) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamDexFundConfigMarketAgents)
		err   error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	for i, tradeToken := range param.TradeTokens {
		if marketInfo, ok := dex.GetMarketInfo(db, tradeToken, param.QuoteTokens[i]); !ok {
			return handleDexReceiveErr(fundLogger, md.MethodName, dex.TradeMarketNotExistsErr, sendBlock)
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

type MethodDexFundPlaceAgentOrder struct {
	MethodName string
}

func (md *MethodDexFundPlaceAgentOrder) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundPlaceAgentOrder) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundPlaceAgentOrder) GetSendQuota(data []byte, gasTable *util.GasTable) (uint64, error) {
	return util.TxGasCost(data, gasTable)
}

func (md *MethodDexFundPlaceAgentOrder) GetReceiveQuota(gasTable *util.GasTable) uint64 {
	return gasTable.DexFundNewAgentOrderGas
}

func (md *MethodDexFundPlaceAgentOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(dex.ParamDexFundNewAgentOrder)
	if err := cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	return dex.PreCheckOrderParam(&param.ParamDexFundNewOrder, true)
}

func (md MethodDexFundPlaceAgentOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamDexFundNewAgentOrder)
		err   error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if blocks, err := dex.DoNewOrder(db, &param.ParamDexFundNewOrder, &param.Principal, &sendBlock.AccountAddress, sendBlock.Hash); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return blocks, nil
	}
}

func handleDexReceiveErr(logger log15.Logger, method string, err error, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	logger.Error("dex receive with err", "error", err.Error(), "method", method, "sendBlockHash", sendBlock.Hash.String(), "sendAddress", sendBlock.AccountAddress.String())
	return nil, err
}
