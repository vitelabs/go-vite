package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

var (
	delegate          = true
	noDelegate        = false
	noDelegateAddress = types.ZERO_ADDRESS
	noBid             = uint8(0)
)

type MethodPledge struct {
	MethodName string
}

func (p *MethodPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodPledge) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodPledge) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.StakeQuota, nil
}
func (p *MethodPledge) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if !util.IsViteToken(block.TokenId) ||
		block.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	beneficialAddr := new(types.Address)
	if err := abi.ABIQuota.UnpackMethod(beneficialAddr, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIQuota.PackMethod(p.MethodName, *beneficialAddr)
	return nil
}
func (p *MethodPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	beneficialAddr := new(types.Address)
	abi.ABIQuota.UnpackMethod(beneficialAddr, p.MethodName, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, sendBlock.AccountAddress, *beneficialAddr, noDelegate, noDelegateAddress, noBid, block.Height)
	var amount *big.Int
	if oldPledge != nil {
		amount = oldPledge.Amount
	} else {
		amount = big.NewInt(0)
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, amount, getPledgeWithdrawHeight(vm, nodeConfig.params.PledgeHeight), beneficialAddr, noDelegate, noDelegateAddress, noBid)
	util.SetValue(db, pledgeKey, pledgeInfo)

	beneficialKey := abi.GetStakeBeneficialKey(*beneficialAddr)
	oldBeneficialData := util.GetValue(db, beneficialKey)
	var beneficialAmount *big.Int
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(abi.VariableStakeBeneficial)
		abi.ABIQuota.UnpackVariable(oldBeneficial, abi.VariableNameStakeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	} else {
		beneficialAmount = big.NewInt(0)
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, beneficialAmount)
	util.SetValue(db, beneficialKey, beneficialData)
	return nil, nil
}

func getPledgeInfo(db vm_db.VmDb, pledgeAddr types.Address, beneficialAddr types.Address, agent bool, agentAddr types.Address, bid uint8, currentIndex uint64) ([]byte, *types.StakeInfo) {
	iterator, err := db.NewStorageIterator(abi.GetStakeInfoKeyPrefix(pledgeAddr))
	util.DealWithErr(err)
	defer iterator.Release()
	maxIndex := uint64(0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				util.DealWithErr(iterator.Error())
			}
			break
		}
		if !abi.IsStakeInfoKey(iterator.Key()) {
			continue
		}
		pledgeInfo := new(types.StakeInfo)
		abi.ABIQuota.UnpackVariable(pledgeInfo, abi.VariableNameStakeInfo, iterator.Value())
		if pledgeInfo.Beneficiary == beneficialAddr && pledgeInfo.IsDelegated == agent &&
			pledgeInfo.DelegateAddress == agentAddr && pledgeInfo.Bid == bid {
			return iterator.Key(), pledgeInfo
		}
		maxIndex = helper.Max(maxIndex, abi.GetIndexFromStakeInfoKey(iterator.Key()))
	}
	if maxIndex < currentIndex {
		return abi.GetStakeInfoKey(pledgeAddr, currentIndex), nil
	}
	return abi.GetStakeInfoKey(pledgeAddr, maxIndex+1), nil
}

func getPledgeWithdrawHeight(vm vmEnvironment, height uint64) uint64 {
	return vm.GlobalStatus().SnapshotBlock().Height + height
}

func pledgeNotDue(oldPledge *types.StakeInfo, vm vmEnvironment) bool {
	return oldPledge.ExpirationHeight > vm.GlobalStatus().SnapshotBlock().Height
}

type MethodCancelPledge struct {
	MethodName string
}

func (p *MethodCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelPledge) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodCancelPledge) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.CancelStakeQuota, nil
}
func (p *MethodCancelPledge) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodCancelPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelStake)
	if err := abi.ABIQuota.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIQuota.PackMethod(p.MethodName, param.Beneficiary, param.Amount)
	return nil
}

func (p *MethodCancelPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamCancelStake)
	abi.ABIQuota.UnpackMethod(param, p.MethodName, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, sendBlock.AccountAddress, param.Beneficiary, noDelegate, noDelegateAddress, noBid, block.Height)
	if oldPledge == nil || pledgeNotDue(oldPledge, vm) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	if oldPledge.Amount.Sign() != 0 && oldPledge.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	beneficialKey := abi.GetStakeBeneficialKey(param.Beneficiary)
	v := util.GetValue(db, beneficialKey)
	oldBeneficial := new(abi.VariableStakeBeneficial)
	err := abi.ABIQuota.UnpackVariable(oldBeneficial, abi.VariableNameStakeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		util.SetValue(db, pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, oldPledge.Amount, oldPledge.ExpirationHeight, oldPledge.Beneficiary, noDelegate, noDelegateAddress, noBid)
		util.SetValue(db, pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		util.SetValue(db, beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, oldBeneficial.Amount)
		util.SetValue(db, beneficialKey, pledgeBeneficial)
	}
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         param.Amount,
			TokenId:        ledger.ViteTokenId,
			Data:           []byte{},
		},
	}, nil
}

type MethodAgentPledge struct {
	MethodName string
}

func (p *MethodAgentPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodAgentPledge) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	param := new(abi.ParamDelegateStake)
	abi.ABIQuota.UnpackMethod(param, p.MethodName, sendBlock.Data)
	callbackData, _ := abi.ABIQuota.PackCallback(p.MethodName, param.StakeAddress, param.Beneficiary, sendBlock.Amount, param.Bid, false)
	return callbackData, true
}

func (p *MethodAgentPledge) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.DelegateStakeQuota, nil
}
func (p *MethodAgentPledge) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodAgentPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if !util.IsViteToken(block.TokenId) ||
		block.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamDelegateStake)
	if err := abi.ABIQuota.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.StakeHeight < nodeConfig.params.PledgeHeight || param.StakeHeight > stakeHeightMax {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIQuota.PackMethod(p.MethodName, param.StakeAddress, param.Beneficiary, param.Bid, param.StakeHeight)
	return nil
}
func (p *MethodAgentPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamDelegateStake)
	abi.ABIQuota.UnpackMethod(param, p.MethodName, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, param.StakeAddress, param.Beneficiary, delegate, sendBlock.AccountAddress, param.Bid, block.Height)
	var amount *big.Int
	oldWithdrawHeight := uint64(0)
	if oldPledge != nil {
		amount = oldPledge.Amount
		oldWithdrawHeight = oldPledge.ExpirationHeight
	} else {
		amount = big.NewInt(0)
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, amount, helper.Max(oldWithdrawHeight, getPledgeWithdrawHeight(vm, param.StakeHeight)), param.Beneficiary, delegate, sendBlock.AccountAddress, param.Bid)
	util.SetValue(db, pledgeKey, pledgeInfo)

	beneficialKey := abi.GetStakeBeneficialKey(param.Beneficiary)
	oldBeneficialData := util.GetValue(db, beneficialKey)
	var beneficialAmount *big.Int
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(abi.VariableStakeBeneficial)
		abi.ABIQuota.UnpackVariable(oldBeneficial, abi.VariableNameStakeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	} else {
		beneficialAmount = big.NewInt(0)
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, beneficialAmount)
	util.SetValue(db, beneficialKey, beneficialData)

	callbackData, _ := abi.ABIQuota.PackCallback(p.MethodName, param.StakeAddress, param.Beneficiary, sendBlock.Amount, param.Bid, true)
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         big.NewInt(0),
			TokenId:        ledger.ViteTokenId,
			Data:           callbackData,
		},
	}, nil
}

type MethodAgentCancelPledge struct {
	MethodName string
}

func (p *MethodAgentCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodAgentCancelPledge) GetRefundData(sendBlock *ledger.AccountBlock, sbHeight uint64) ([]byte, bool) {
	param := new(abi.ParamCancelDelegateStake)
	abi.ABIQuota.UnpackMethod(param, p.MethodName, sendBlock.Data)
	callbackData, _ := abi.ABIQuota.PackCallback(p.MethodName, param.StakeAddress, param.Beneficiary, param.Amount, param.Bid, false)
	return callbackData, true
}

func (p *MethodAgentCancelPledge) GetSendQuota(data []byte, gasTable *util.QuotaTable) (uint64, error) {
	return gasTable.CancelDelegateStakeQuota, nil
}

func (p *MethodAgentCancelPledge) GetReceiveQuota(gasTable *util.QuotaTable) uint64 {
	return 0
}

func (p *MethodAgentCancelPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelDelegateStake)
	if err := abi.ABIQuota.UnpackMethod(param, p.MethodName, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIQuota.PackMethod(p.MethodName, param.StakeAddress, param.Beneficiary, param.Amount, param.Bid)
	return nil
}

func (p *MethodAgentCancelPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamCancelDelegateStake)
	abi.ABIQuota.UnpackMethod(param, p.MethodName, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, param.StakeAddress, param.Beneficiary, delegate, sendBlock.AccountAddress, param.Bid, block.Height)
	if oldPledge == nil || pledgeNotDue(oldPledge, vm) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	if oldPledge.Amount.Sign() != 0 && oldPledge.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	oldBeneficial := new(abi.VariableStakeBeneficial)
	beneficialKey := abi.GetStakeBeneficialKey(param.Beneficiary)
	v := util.GetValue(db, beneficialKey)
	err := abi.ABIQuota.UnpackVariable(oldBeneficial, abi.VariableNameStakeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		util.SetValue(db, pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeInfo, oldPledge.Amount, oldPledge.ExpirationHeight, oldPledge.Beneficiary, delegate, oldPledge.DelegateAddress, param.Bid)
		util.SetValue(db, pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		util.SetValue(db, beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIQuota.PackVariable(abi.VariableNameStakeBeneficial, oldBeneficial.Amount)
		util.SetValue(db, beneficialKey, pledgeBeneficial)
	}

	callbackData, _ := abi.ABIQuota.PackCallback(p.MethodName, param.StakeAddress, param.Beneficiary, param.Amount, param.Bid, true)
	return []*ledger.AccountBlock{
		{
			AccountAddress: block.AccountAddress,
			ToAddress:      sendBlock.AccountAddress,
			BlockType:      ledger.BlockTypeSendCall,
			Amount:         param.Amount,
			TokenId:        ledger.ViteTokenId,
			Data:           callbackData,
		},
	}, nil
}
