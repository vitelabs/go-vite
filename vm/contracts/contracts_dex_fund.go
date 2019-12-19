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

type MethodDexFundDeposit struct {
	MethodName string
}

func (md *MethodDexFundDeposit) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundDeposit) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundDeposit) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundDeposit) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundDepositQuota
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

func (md *MethodDexFundWithdraw) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundWithdraw) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundWithdrawQuota
}

func (md *MethodDexFundWithdraw) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamWithdraw)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	if param.Amount.Sign() <= 0 {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md *MethodDexFundWithdraw) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamWithdraw)
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

func (md *MethodDexFundOpenNewMarket) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundOpenNewMarket) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundOpenNewMarketQuota
}

func (md *MethodDexFundOpenNewMarket) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamOpenNewMarket)
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
	param := new(dex.ParamOpenNewMarket)
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
					ToAddress:      types.AddressAsset,
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

func (md *MethodDexFundPlaceOrder) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundPlaceOrder) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundPlaceOrderQuota
}

func (md *MethodDexFundPlaceOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) (err error) {
	param := new(dex.ParamPlaceOrder)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	return dex.PreCheckOrderParam(param, dex.IsStemFork(db))
}

func (md *MethodDexFundPlaceOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(dex.ParamPlaceOrder)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if blocks, err := dex.DoPlaceOrder(db, param, &sendBlock.AccountAddress, nil, sendBlock.Hash); err != nil {
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

func (md *MethodDexFundSettleOrders) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundSettleOrders) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundSettleOrdersQuota
}

func (md *MethodDexFundSettleOrders) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	if block.AccountAddress != types.AddressDexTrade {
		return dex.InvalidSourceAddressErr
	}
	param := new(dex.ParamSerializedData)
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
	param := new(dex.ParamSerializedData)
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
			dex.SettleOperatorFees(db, vm.ConsensusReader(), settleActions.FeeActions, marketInfo)
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

func (md *MethodDexFundTriggerPeriodJob) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundTriggerPeriodJob) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundTriggerPeriodJobQuota
}

func (md *MethodDexFundTriggerPeriodJob) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamTriggerPeriodJob), md.MethodName, block.Data)
}

func (md MethodDexFundTriggerPeriodJob) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err error
	)
	if !dex.ValidTriggerAddress(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamTriggerPeriodJob)
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
	var blocks []*ledger.AccountBlock
	if param.BizType <= dex.OperatorFeeDividendJob || param.BizType == dex.FinishVxUnlock || param.BizType == dex.FinishCancelMiningStake {
		switch param.BizType {
		case dex.FeeDividendJob:
			blocks, err = dex.DoFeesDividend(db, param.PeriodId)
		case dex.OperatorFeeDividendJob:
			err = dex.DoOperatorFeesDividend(db, param.PeriodId)
		case dex.FinishVxUnlock:
			err = dex.DoFinishVxUnlock(db, param.PeriodId)
		case dex.FinishCancelMiningStake:
			err = dex.DoFinishCancelMiningStake(db, param.PeriodId)
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
				return handleDexReceiveErr(fundLogger, md.MethodName, fmt.Errorf("no vx available on mine for staking"), sendBlock)
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
	return blocks, nil
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

func (md *MethodDexFundStakeForMining) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundStakeForMining) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundStakeForMiningQuota
}

func (md *MethodDexFundStakeForMining) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamStakeForMining)
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else {
		if param.Amount.Cmp(dex.StakeForMiningMinAmount) < 0 {
			return dex.InvalidStakeAmountErr
		}
		if param.ActionType != dex.Stake && param.ActionType != dex.CancelStake {
			return dex.InvalidStakeActionTypeErr
		}
	}
	return nil
}

func (md MethodDexFundStakeForMining) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) (appendBlocks []*ledger.AccountBlock, err error) {
	var param = new(dex.ParamStakeForMining)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	stakeHeight := nodeConfig.params.StakeHeight
	if !dex.IsDexMiningFork(db) && dex.IsEarthFork(db) {
		stakeHeight = 1
	}
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForMining, param.ActionType, sendBlock.AccountAddress, types.ZERO_ADDRESS, param.Amount, stakeHeight, block); err != nil {
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

func (md *MethodDexFundStakeForVIP) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundStakeForVIP) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundStakeForVipQuota
}

func (md *MethodDexFundStakeForVIP) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamStakeForVIP)
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
	var param = new(dex.ParamStakeForVIP)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForVIP, param.ActionType, sendBlock.AccountAddress, types.ZERO_ADDRESS, dex.StakeForVIPAmount, nodeConfig.params.DexVipStakeHeight, block); err != nil {
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

func (md *MethodDexFundStakeForSVIP) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundStakeForSVIP) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundStakeForSuperVIPQuota
}

func (md *MethodDexFundStakeForSVIP) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err   error
		param = new(dex.ParamStakeForVIP)
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
	var param = new(dex.ParamStakeForVIP)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForSuperVIP, param.ActionType, sendBlock.AccountAddress, types.ZERO_ADDRESS, dex.StakeForSuperVIPAmount, nodeConfig.params.DexSuperVipStakeHeight, block); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundStakeForPrincipalSVIP struct {
	MethodName string
}

func (md *MethodDexFundStakeForPrincipalSVIP) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundStakeForPrincipalSVIP) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundStakeForPrincipalSVIP) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundStakeForPrincipalSVIP) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundStakeForPrincipalSuperVIPQuota
}

func (md *MethodDexFundStakeForPrincipalSVIP) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err       error
		principal = new(types.Address)
	)
	if err = cabi.ABIDexFund.UnpackMethod(principal, md.MethodName, block.Data); err != nil {
		return err
	}
	if principal == nil || *principal == types.ZERO_ADDRESS || *principal == block.AccountAddress {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundStakeForPrincipalSVIP) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var principal = new(types.Address)
	cabi.ABIDexFund.UnpackMethod(principal, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.HandleStakeAction(db, dex.StakeForPrincipalSuperVIP, dex.Stake, sendBlock.AccountAddress, *principal, dex.StakeForSuperVIPAmount, nodeConfig.params.DexSuperVipStakeHeight, block); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundCancelStakeById struct {
	MethodName string
}

func (md *MethodDexFundCancelStakeById) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelStakeById) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundCancelStakeById) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundCancelStakeById) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundCancelStakeByIdQuota
}

func (md *MethodDexFundCancelStakeById) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var (
		err error
		id  = new(types.Hash)
	)
	if err = cabi.ABIDexFund.UnpackMethod(id, md.MethodName, block.Data); err != nil {
		return err
	}
	return nil
}

func (md MethodDexFundCancelStakeById) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var id = new(types.Hash)
	cabi.ABIDexFund.UnpackMethod(id, md.MethodName, sendBlock.Data)
	if appendBlocks, err := dex.DoCancelStakeV2(db, sendBlock.AccountAddress, *id); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return appendBlocks, nil
	}
}

type MethodDexFundDelegateStakeCallback struct {
	MethodName string
}

func (md *MethodDexFundDelegateStakeCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundDelegateStakeCallback) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundDelegateStakeCallback) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundDelegateStakeCallback) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundDelegateStakeCallbackQuota
}

func (md *MethodDexFundDelegateStakeCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressQuota {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDelegateStakeCallback), md.MethodName, block.Data)
}

func (md MethodDexFundDelegateStakeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var callbackParam = new(dex.ParamDelegateStakeCallback)
	cabi.ABIDexFund.UnpackMethod(callbackParam, md.MethodName, sendBlock.Data)
	if callbackParam.Success {
		switch callbackParam.Bid {
		case dex.StakeForMining:
			stakedAmount := dex.GetMiningStakedAmount(db, callbackParam.StakeAddress)
			stakedAmount.Add(stakedAmount, callbackParam.Amount)
			dex.SaveMiningStakedAmount(db, callbackParam.StakeAddress, stakedAmount)
			if err := dex.OnMiningStakeSuccess(db, vm.ConsensusReader(), callbackParam.StakeAddress, callbackParam.Amount, stakedAmount); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		case dex.StakeForVIP:
			if vipStaking, ok := dex.GetVIPStaking(db, callbackParam.StakeAddress); ok { //duplicate staking for vip
				vipStaking.StakedTimes = vipStaking.StakedTimes + 1
				dex.SaveVIPStaking(db, callbackParam.StakeAddress, vipStaking)
				// duplicate staking for vip, cancel stake
				return dex.DoCancelStakeV1(db, callbackParam.StakeAddress, callbackParam.Bid, callbackParam.Amount)
			} else {
				vipStaking.Timestamp = dex.GetTimestampInt64(db)
				vipStaking.StakedTimes = 1
				dex.SaveVIPStaking(db, callbackParam.StakeAddress, vipStaking)
			}
		case dex.StakeForSuperVIP:
			if superVIPStaking, ok := dex.GetSuperVIPStaking(db, callbackParam.StakeAddress); ok { //duplicate staking for super vip
				superVIPStaking.StakedTimes = superVIPStaking.StakedTimes + 1
				dex.SaveSuperVIPStaking(db, callbackParam.StakeAddress, superVIPStaking)
				// duplicate staking for vip, cancel stake
				return dex.DoCancelStakeV1(db, callbackParam.StakeAddress, callbackParam.Bid, callbackParam.Amount)
			} else {
				superVIPStaking.Timestamp = dex.GetTimestampInt64(db)
				superVIPStaking.StakedTimes = 1
				dex.SaveSuperVIPStaking(db, callbackParam.StakeAddress, superVIPStaking)
			}
		}
	} else {
		switch callbackParam.Bid {
		case dex.StakeForMining:
			if callbackParam.Amount.Cmp(sendBlock.Amount) != 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			}
		case dex.StakeForVIP:
			if dex.StakeForVIPAmount.Cmp(sendBlock.Amount) != 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			}
		case dex.StakeForSuperVIP:
			if dex.StakeForSuperVIPAmount.Cmp(sendBlock.Amount) != 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			}
		}
		dex.DepositAccount(db, callbackParam.StakeAddress, ledger.ViteTokenId, sendBlock.Amount)
	}
	return nil, nil
}

type MethodDexFundCancelDelegateStakeCallback struct {
	MethodName string
}

func (md *MethodDexFundCancelDelegateStakeCallback) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelDelegateStakeCallback) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundCancelDelegateStakeCallback) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundCancelDelegateStakeCallback) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundCancelDelegateStakeCallbackQuota
}

func (md *MethodDexFundCancelDelegateStakeCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressQuota {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDelegateStakeCallback), md.MethodName, block.Data)
}

func (md MethodDexFundCancelDelegateStakeCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDelegateStakeCallback)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	if param.Success {
		switch param.Bid {
		case dex.StakeForMining:
			if param.Amount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			stakedAmount := dex.GetMiningStakedAmount(db, param.StakeAddress)
			leaved := new(big.Int).Sub(stakedAmount, sendBlock.Amount)
			if leaved.Sign() < 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			} else if leaved.Sign() == 0 {
				dex.DeleteMiningStakedAmount(db, param.StakeAddress)
			} else {
				dex.SaveMiningStakedAmount(db, param.StakeAddress, leaved)
			}
			if err := dex.OnCancelMiningStakeSuccess(db, vm.ConsensusReader(), param.StakeAddress, sendBlock.Amount, leaved); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		case dex.StakeForVIP:
			if dex.StakeForVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			if vipStaking, ok := dex.GetVIPStaking(db, param.StakeAddress); ok {
				vipStaking.StakedTimes = vipStaking.StakedTimes - 1
				if vipStaking.StakedTimes == 0 {
					dex.DeleteVIPStaking(db, param.StakeAddress)
				} else {
					dex.SaveVIPStaking(db, param.StakeAddress, vipStaking)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.VIPStakingNotExistsErr, sendBlock)
			}
		case dex.StakeForSuperVIP:
			if dex.StakeForSuperVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			if superVIPStaking, ok := dex.GetSuperVIPStaking(db, param.StakeAddress); ok {
				superVIPStaking.StakedTimes = superVIPStaking.StakedTimes - 1
				if superVIPStaking.StakedTimes == 0 {
					dex.DeleteSuperVIPStaking(db, param.StakeAddress)
				} else {
					dex.SaveSuperVIPStaking(db, param.StakeAddress, superVIPStaking)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.SuperVIPStakingNotExistsErr, sendBlock)
			}
		}
		if dex.IsEarthFork(db) && param.Bid == dex.StakeForMining {
			dex.ScheduleCancelStake(db, param.StakeAddress, sendBlock.Amount)
		} else {
			dex.DepositAccount(db, param.StakeAddress, ledger.ViteTokenId, sendBlock.Amount)
		}
	}
	return nil, nil
}

type MethodDexFundDelegateStakeCallbackV2 struct {
	MethodName string
}

func (md *MethodDexFundDelegateStakeCallbackV2) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundDelegateStakeCallbackV2) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundDelegateStakeCallbackV2) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundDelegateStakeCallbackV2) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundDelegateStakeCallbackV2Quota
}

func (md *MethodDexFundDelegateStakeCallbackV2) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressQuota {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDelegateStakeCallbackV2), md.MethodName, block.Data)
}

func (md MethodDexFundDelegateStakeCallbackV2) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) (blocks []*ledger.AccountBlock, err error) {
	var param = new(dex.ParamDelegateStakeCallbackV2)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	var (
		info *dex.DelegateStakeInfo
		ok   bool
	)
	if info, ok = dex.GetDelegateStakeInfo(db, param.Id.Bytes()); !ok {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.StakingInfoByIdNotExistsErr, sendBlock)
	}
	address, _ := types.BytesToAddress(info.Address)
	amount := new(big.Int).SetBytes(info.Amount)
	if param.Success {
		switch info.StakeType {
		case dex.StakeForMining:
			stakedAmount := dex.GetMiningStakedV2Amount(db, address)
			stakedAmount.Add(stakedAmount, amount)
			dex.SaveMiningStakedV2Amount(db, address, stakedAmount)
			if err = dex.OnMiningStakeSuccessV2(db, vm.ConsensusReader(), address, amount, stakedAmount); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		case dex.StakeForVIP:
			if vipStaking, ok := dex.GetVIPStaking(db, address); ok { //duplicate staking for vip
				vipStaking.StakedTimes = vipStaking.StakedTimes + 1
				vipStaking.StakingHashes = append(vipStaking.StakingHashes, param.Id.Bytes())
				dex.SaveVIPStaking(db, address, vipStaking)
				// duplicate staking for vip, cancel stake
				blocks, err = dex.DoRawCancelStakeV2(param.Id)
			} else {
				vipStaking.Timestamp = dex.GetTimestampInt64(db)
				vipStaking.StakedTimes = 1
				vipStaking.StakingHashes = append(vipStaking.StakingHashes, param.Id.Bytes())
				dex.SaveVIPStaking(db, address, vipStaking)
			}
		case dex.StakeForSuperVIP, dex.StakeForPrincipalSuperVIP:
			if info.StakeType == dex.StakeForPrincipalSuperVIP {
				address, _ = types.BytesToAddress(info.Principal) // principal
			}
			if superVIPStaking, ok := dex.GetSuperVIPStaking(db, address); ok { //duplicate staking for super vip
				superVIPStaking.StakedTimes = superVIPStaking.StakedTimes + 1
				superVIPStaking.StakingHashes = append(superVIPStaking.StakingHashes, param.Id.Bytes())
				dex.SaveSuperVIPStaking(db, address, superVIPStaking)
				// duplicate staking for vip, cancel stake
				blocks, err = dex.DoRawCancelStakeV2(param.Id)
			} else {
				superVIPStaking.Timestamp = dex.GetTimestampInt64(db)
				superVIPStaking.StakedTimes = 1
				superVIPStaking.StakingHashes = append(superVIPStaking.StakingHashes, param.Id.Bytes())
				dex.SaveSuperVIPStaking(db, address, superVIPStaking)
			}
		}
		serialNo := dex.SaveDelegateStakeAddressIndex(db, param.Id, info.StakeType, info.Address)
		dex.ConfirmDelegateStakeInfo(db, param.Id, info, serialNo)
	} else {
		switch info.StakeType {
		case dex.StakeForMining:
			if bytes.Equal(info.Amount, sendBlock.Amount.Bytes()) {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			}
		case dex.StakeForVIP:
			if dex.StakeForVIPAmount.Cmp(sendBlock.Amount) != 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			}
		case dex.StakeForSuperVIP, dex.StakeForPrincipalSuperVIP:
			if dex.StakeForSuperVIPAmount.Cmp(sendBlock.Amount) != 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			}
		}
		dex.DepositAccount(db, address, ledger.ViteTokenId, sendBlock.Amount)
		dex.DeleteDelegateStakeInfo(db, param.Id.Bytes())
	}
	return
}

type MethodDexFundCancelDelegateStakeCallbackV2 struct {
	MethodName string
}

func (md *MethodDexFundCancelDelegateStakeCallbackV2) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundCancelDelegateStakeCallbackV2) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundCancelDelegateStakeCallbackV2) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundCancelDelegateStakeCallbackV2) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundDelegateCancelStakeCallbackV2Quota
}

func (md *MethodDexFundCancelDelegateStakeCallbackV2) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressQuota {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDelegateStakeCallbackV2), md.MethodName, block.Data)
}

func (md MethodDexFundCancelDelegateStakeCallbackV2) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var param = new(dex.ParamDelegateStakeCallbackV2)
	cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data)
	var (
		info *dex.DelegateStakeInfo
		ok   bool
	)
	if info, ok = dex.GetDelegateStakeInfo(db, param.Id.Bytes()); !ok {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.StakingInfoByIdNotExistsErr, sendBlock)
	}
	address, _ := types.BytesToAddress(info.Address)
	if param.Success {
		switch info.StakeType {
		case dex.StakeForMining:
			if !bytes.Equal(info.Amount, sendBlock.Amount.Bytes()) {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			stakedAmount := dex.GetMiningStakedV2Amount(db, address)
			leaved := new(big.Int).Sub(stakedAmount, sendBlock.Amount)
			if leaved.Sign() < 0 {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidAmountForStakeCallbackErr, sendBlock)
			} else if leaved.Sign() == 0 {
				dex.DeleteMiningStakedV2Amount(db, address)
			} else {
				dex.SaveMiningStakedV2Amount(db, address, leaved)
			}
			if err := dex.OnCancelMiningStakeSuccessV2(db, vm.ConsensusReader(), address, sendBlock.Amount, leaved); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
		case dex.StakeForVIP:
			if dex.StakeForVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			if vipStaking, ok := dex.GetVIPStaking(db, address); ok {
				vipStaking.StakedTimes = vipStaking.StakedTimes - 1
				if vipStaking.StakedTimes == 0 {
					dex.DeleteVIPStaking(db, address)
				} else {
					if ok = dex.ReduceVipStakingHash(vipStaking, param.Id); !ok {
						panic(dex.InvalidIdForStakeCallbackErr)
					}
					dex.SaveVIPStaking(db, address, vipStaking)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.VIPStakingNotExistsErr, sendBlock)
			}
		case dex.StakeForSuperVIP, dex.StakeForPrincipalSuperVIP:
			if info.StakeType == dex.StakeForPrincipalSuperVIP {
				address, _ = types.BytesToAddress(info.Principal) // principal
			}
			if dex.StakeForSuperVIPAmount.Cmp(sendBlock.Amount) != 0 {
				panic(dex.InvalidAmountForStakeCallbackErr)
			}
			if superVIPStaking, ok := dex.GetSuperVIPStaking(db, address); ok {
				superVIPStaking.StakedTimes = superVIPStaking.StakedTimes - 1
				if superVIPStaking.StakedTimes == 0 {
					dex.DeleteSuperVIPStaking(db, address)
				} else {
					if ok = dex.ReduceVipStakingHash(superVIPStaking, param.Id); !ok {
						panic(dex.InvalidIdForStakeCallbackErr)
					}
					dex.SaveSuperVIPStaking(db, address, superVIPStaking)
				}
			} else {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.SuperVIPStakingNotExistsErr, sendBlock)
			}
		}
		if info.StakeType == dex.StakeForPrincipalSuperVIP {
			stakeAddress, _ := types.BytesToAddress(info.Address) // address
			dex.DepositAccount(db, stakeAddress, ledger.ViteTokenId, sendBlock.Amount)
		} else if info.StakeType == dex.StakeForMining {
			dex.ScheduleCancelStake(db, address, sendBlock.Amount)
		} else {
			dex.DepositAccount(db, address, ledger.ViteTokenId, sendBlock.Amount)
		}
		dex.DeleteDelegateStakeInfo(db, param.Id.Bytes())
		dex.DeleteDelegateStakeAddressIndex(db, info.Address, info.SerialNo)
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

func (md *MethodDexFundGetTokenInfoCallback) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundGetTokenInfoCallback) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundGetTokenInfoCallbackQuota
}

func (md *MethodDexFundGetTokenInfoCallback) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.AccountAddress != types.AddressAsset {
		return dex.InvalidSourceAddressErr
	}
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamGetTokenInfoCallback), md.MethodName, block.Data)
}

func (md MethodDexFundGetTokenInfoCallback) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var callbackParam = new(dex.ParamGetTokenInfoCallback)
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

func (md *MethodDexFundDexAdminConfig) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundDexAdminConfig) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundAdminConfigQuota
}

func (md *MethodDexFundDexAdminConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamDexAdminConfig), md.MethodName, block.Data)
}

func (md MethodDexFundDexAdminConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var err error
	var param = new(dex.ParamDexAdminConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.AdminConfigOwner) {
			dex.SetOwner(db, param.Owner)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.AdminConfigTimeOracle) {
			dex.SetTimeOracle(db, param.TimeOracle)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.AdminConfigPeriodJobTrigger) {
			dex.SetPeriodJobTrigger(db, param.PeriodJobTrigger)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.AdminConfigStopDex) {
			dex.SaveDexStopped(db, param.StopDex)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.AdminConfigMakerMiningAdmin) {
			dex.SaveMakerMiningAdmin(db, param.MakerMiningAdmin)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.AdminConfigMaintainer) {
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

func (md *MethodDexFundTradeAdminConfig) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundTradeAdminConfig) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundTradeAdminConfigQuota
}

func (md *MethodDexFundTradeAdminConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamTradeAdminConfig), md.MethodName, block.Data)
}

func (md MethodDexFundTradeAdminConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) (blocks []*ledger.AccountBlock, err error) {
	var param = new(dex.ParamTradeAdminConfig)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if dex.IsOwner(db, sendBlock.AccountAddress) {
		if dex.IsOperationValidWithMask(param.OperationCode, dex.TradeAdminConfigMineMarket) {
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
		if dex.IsOperationValidWithMask(param.OperationCode, dex.TradeAdminConfigNewQuoteToken) {
			if param.QuoteTokenType < dex.ViteTokenType || param.QuoteTokenType > dex.UsdTokenType {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidQuoteTokenTypeErr, sendBlock)
			} else {
				if _, ok := dex.QuoteTokenTypeInfos[int32(param.QuoteTokenType)]; !ok {
					return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidQuoteTokenTypeErr, sendBlock)
				}
			}
			if tokenInfo, ok := dex.GetTokenInfo(db, param.NewQuoteToken); !ok {
				getTokenInfoData := dex.OnSetQuoteTokenPending(db, param.NewQuoteToken, param.QuoteTokenType)
				return []*ledger.AccountBlock{
					{
						AccountAddress: types.AddressDexFund,
						ToAddress:      types.AddressAsset,
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
		if dex.IsOperationValidWithMask(param.OperationCode, dex.TradeAdminConfigTradeThreshold) {
			dex.SaveTradeThreshold(db, param.TokenTypeForTradeThreshold, param.MinTradeThreshold)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.TradeAdminConfigMineThreshold) {
			dex.SaveMineThreshold(db, param.TokenTypeForMiningThreshold, param.MinMiningThreshold)
		}
		if dex.IsEarthFork(db) && dex.IsOperationValidWithMask(param.OperationCode, dex.TradeAdminStartNormalMine) && !dex.IsNormalMiningStarted(db) {
			dex.StartNormalMine(db)
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.TradeAdminBurnExtraVx) && dex.IsNormalMiningStarted(db) && dex.GetVxBurnAmount(db).Sign() == 0 {
			if blocks, err = dex.BurnExtraVx(db); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			} else {
				return
			}
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

func (md *MethodDexFundMarketAdminConfig) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundMarketAdminConfig) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundMarketAdminConfigQuota
}

func (md *MethodDexFundMarketAdminConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamMarketAdminConfig), md.MethodName, block.Data)
}

func (md MethodDexFundMarketAdminConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err        error
		marketInfo *dex.MarketInfo
		ok         bool
	)
	var param = new(dex.ParamMarketAdminConfig)
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
			if !dex.ValidOperatorFeeRate(param.TakerFeeRate) {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidOperatorFeeRateErr, sendBlock)
			}
			marketInfo.TakerOperatorFeeRate = param.TakerFeeRate
		}
		if dex.IsOperationValidWithMask(param.OperationCode, dex.MarketOwnerConfigMakerRate) {
			if !dex.ValidOperatorFeeRate(param.MakerFeeRate) {
				return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidOperatorFeeRateErr, sendBlock)
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

func (md *MethodDexFundTransferTokenOwnership) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundTransferTokenOwnership) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundTransferTokenOwnershipQuota
}

func (md *MethodDexFundTransferTokenOwnership) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamTransferTokenOwnership), md.MethodName, block.Data)
}

func (md MethodDexFundTransferTokenOwnership) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err   error
		param = new(dex.ParamTransferTokenOwnership)
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
				ToAddress:      types.AddressAsset,
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

func (md *MethodDexFundNotifyTime) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundNotifyTime) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundNotifyTimeQuota
}

func (md *MethodDexFundNotifyTime) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	return cabi.ABIDexFund.UnpackMethod(new(dex.ParamNotifyTime), md.MethodName, block.Data)
}

func (md MethodDexFundNotifyTime) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		err             error
		notifyTimeParam = new(dex.ParamNotifyTime)
	)
	if !dex.ValidTimeOracle(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidSourceAddressErr, sendBlock)
	}
	if err = cabi.ABIDexFund.UnpackMethod(notifyTimeParam, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if err = dex.SetDexTimestamp(db, notifyTimeParam.Timestamp, vm.ConsensusReader()); err != nil {
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

func (md *MethodDexFundCreateNewInviter) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundCreateNewInviter) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundCreateNewInviterQuota
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

func (md *MethodDexFundBindInviteCode) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundBindInviteCode) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundBindInviteCodeQuota
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

func (md *MethodDexFundEndorseVx) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundEndorseVx) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundEndorseVxQuota
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

func (md *MethodDexFundSettleMakerMinedVx) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundSettleMakerMinedVx) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundSettleMakerMinedVxQuota
}

func (md *MethodDexFundSettleMakerMinedVx) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var err error
	param := new(dex.ParamSerializedData)
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
	if !dex.IsMakerMiningAdmin(db, sendBlock.AccountAddress) {
		return handleDexReceiveErr(fundLogger, md.MethodName, dex.InvalidSourceAddressErr, sendBlock)
	}
	param := new(dex.ParamSerializedData)
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
	if poolAmt = dex.GetMakerMiningPoolByPeriodId(db, actions.Period); poolAmt.Sign() == 0 {
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
			if err = dex.OnVxMined(db, vm.ConsensusReader(), addr, amt); err != nil {
				return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
			}
			poolAmt.Sub(poolAmt, amt)
			if poolAmt.Sign() <= 0 {
				break
			}
		}
	}
	if poolAmt.Sign() > 0 {
		dex.SaveMakerMiningPoolByPeriodId(db, actions.Period, poolAmt)
		dex.SaveLastSettledMakerMinedVxPage(db, actions.Page)
	} else {
		finish = true
		dex.DeleteMakerMiningPoolByPeriodId(db, actions.Period)
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

func (md *MethodDexFundConfigMarketAgents) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundConfigMarketAgents) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFundConfigMarketAgentsQuota
}

func (md *MethodDexFundConfigMarketAgents) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	var param = new(dex.ParamConfigMarketAgents)
	if err := cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else if param.ActionType != dex.GrantAgent && param.ActionType != dex.RevokeAgent || len(param.TradeTokens) == 0 || len(param.TradeTokens) != len(param.QuoteTokens) || block.AccountAddress == param.Agent {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundConfigMarketAgents) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamConfigMarketAgents)
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

func (md *MethodDexFundPlaceAgentOrder) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundPlaceAgentOrder) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFunPlaceAgentOrderQuota
}

func (md *MethodDexFundPlaceAgentOrder) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(dex.ParamPlaceAgentOrder)
	if err := cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	}
	return dex.PreCheckOrderParam(&param.ParamPlaceOrder, true)
}

func (md MethodDexFundPlaceAgentOrder) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamPlaceAgentOrder)
		err   error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	if blocks, err := dex.DoPlaceOrder(db, &param.ParamPlaceOrder, &param.Principal, &sendBlock.AccountAddress, sendBlock.Hash); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	} else {
		return blocks, nil
	}
}

type MethodDexFundLockVxForDividend struct {
	MethodName string
}

func (md *MethodDexFundLockVxForDividend) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundLockVxForDividend) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundLockVxForDividend) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundLockVxForDividend) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFunLockVxForDividendQuota
}

func (md *MethodDexFundLockVxForDividend) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(dex.ParamLockVxForDividend)
	if err := cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else if param.ActionType != dex.LockVx && param.ActionType != dex.UnlockVx || param.Amount.Cmp(dex.VxLockThreshold) < 0 {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundLockVxForDividend) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param            = new(dex.ParamLockVxForDividend)
		updatedVxAccount *dexproto.Account
		err              error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	switch param.ActionType {
	case dex.LockVx:
		if updatedVxAccount, err = dex.LockVxForDividend(db, sendBlock.AccountAddress, param.Amount); err != nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
		} else {
			dex.DoSettleVxFunds(db, vm.ConsensusReader(), sendBlock.AccountAddress.Bytes(), param.Amount, updatedVxAccount)
		}
	case dex.UnlockVx:
		if updatedVxAccount, err = dex.ScheduleVxUnlockForDividend(db, sendBlock.AccountAddress, param.Amount); err != nil {
			return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
		} else {
			dex.AddVxUnlock(db, vm.ConsensusReader(), sendBlock.AccountAddress, param.Amount)
			dex.DoSettleVxFunds(db, vm.ConsensusReader(), sendBlock.AccountAddress.Bytes(), new(big.Int).Neg(param.Amount), updatedVxAccount)
		}
	}
	return nil, nil
}

type MethodDexFundSwitchConfig struct {
	MethodName string
}

func (md *MethodDexFundSwitchConfig) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (md *MethodDexFundSwitchConfig) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (md *MethodDexFundSwitchConfig) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return util.RequestQuotaCost(data, gasTable)
}

func (md *MethodDexFundSwitchConfig) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return gasTable.DexFunSwitchConfigQuota
}

func (md *MethodDexFundSwitchConfig) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	param := new(dex.ParamSwitchConfig)
	if err := cabi.ABIDexFund.UnpackMethod(param, md.MethodName, block.Data); err != nil {
		return err
	} else if param.SwitchType != dex.AutoLockMinedVx {
		return dex.InvalidInputParamErr
	}
	return nil
}

func (md MethodDexFundSwitchConfig) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	var (
		param = new(dex.ParamSwitchConfig)
		err   error
	)
	if err = cabi.ABIDexFund.UnpackMethod(param, md.MethodName, sendBlock.Data); err != nil {
		return handleDexReceiveErr(fundLogger, md.MethodName, err, sendBlock)
	}
	switch param.SwitchType {
	case dex.AutoLockMinedVx:
		dex.SetAutoLockMinedVx(db, sendBlock.AccountAddress.Bytes(), param.Enable)
	}
	return nil, nil
}

func handleDexReceiveErr(logger log15.Logger, method string, err error, sendBlock *ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	logger.Error("dex receive with err", "error", err.Error(), "method", method, "sendBlockHash", sendBlock.Hash.String(), "sendAddress", sendBlock.AccountAddress.String())
	return nil, err
}
