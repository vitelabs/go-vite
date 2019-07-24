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
	Agent          = true
	NoAgent        = false
	NoAgentAddress = types.ZERO_ADDRESS
	NoBid          = uint8(0)
)

type MethodPledge struct{}

func (p *MethodPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodPledge) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodPledge) GetSendQuota(data []byte) (uint64, error) {
	return PledgeGas, nil
}
func (p *MethodPledge) GetReceiveQuota() uint64 {
	return 0
}

func (p *MethodPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if !util.IsViteToken(block.TokenId) ||
		block.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	beneficialAddr := new(types.Address)
	if err := abi.ABIPledge.UnpackMethod(beneficialAddr, abi.MethodNamePledge, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIPledge.PackMethod(abi.MethodNamePledge, *beneficialAddr)
	return nil
}
func (p *MethodPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	beneficialAddr := new(types.Address)
	abi.ABIPledge.UnpackMethod(beneficialAddr, abi.MethodNamePledge, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, sendBlock.AccountAddress, *beneficialAddr, NoAgent, NoAgentAddress, NoBid, block.Height)
	var amount *big.Int
	if oldPledge != nil {
		amount = oldPledge.Amount
	} else {
		amount = big.NewInt(0)
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, amount, getPledgeWithdrawHeight(vm, nodeConfig.params.PledgeHeight), beneficialAddr, NoAgent, NoAgentAddress, NoBid)
	util.SetValue(db, pledgeKey, pledgeInfo)

	beneficialKey := abi.GetPledgeBeneficialKey(*beneficialAddr)
	oldBeneficialData := util.GetValue(db, beneficialKey)
	var beneficialAmount *big.Int
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(abi.VariablePledgeBeneficial)
		abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	} else {
		beneficialAmount = big.NewInt(0)
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, beneficialAmount)
	util.SetValue(db, beneficialKey, beneficialData)
	return nil, nil
}

func getPledgeInfo(db vm_db.VmDb, pledgeAddr types.Address, beneficialAddr types.Address, agent bool, agentAddr types.Address, bid uint8, currentIndex uint64) ([]byte, *abi.PledgeInfo) {
	iterator, err := db.NewStorageIterator(abi.GetPledgeKeyPrefix(pledgeAddr))
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
		if !abi.IsPledgeKey(iterator.Key()) {
			continue
		}
		pledgeInfo := new(abi.PledgeInfo)
		abi.ABIPledge.UnpackVariable(pledgeInfo, abi.VariableNamePledgeInfo, iterator.Value())
		if pledgeInfo.BeneficialAddr == beneficialAddr && pledgeInfo.Agent == agent &&
			pledgeInfo.AgentAddress == agentAddr && pledgeInfo.Bid == bid {
			return iterator.Key(), pledgeInfo
		}
		maxIndex = helper.Max(maxIndex, abi.GetIndexFromPledgeKey(iterator.Key()))
	}
	if maxIndex < currentIndex {
		return abi.GetPledgeKey(pledgeAddr, currentIndex), nil
	} else {
		return abi.GetPledgeKey(pledgeAddr, maxIndex+1), nil
	}
}

func getPledgeWithdrawHeight(vm vmEnvironment, height uint64) uint64 {
	return vm.GlobalStatus().SnapshotBlock().Height + height
}

func pledgeNotDue(oldPledge *abi.PledgeInfo, vm vmEnvironment) bool {
	return oldPledge.WithdrawHeight > vm.GlobalStatus().SnapshotBlock().Height
}

type MethodCancelPledge struct{}

func (p *MethodCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelPledge) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	return []byte{}, false
}

func (p *MethodCancelPledge) GetSendQuota(data []byte) (uint64, error) {
	return CancelPledgeGas, nil
}
func (p *MethodCancelPledge) GetReceiveQuota() uint64 {
	return 0
}

func (p *MethodCancelPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelPledge)
	if err := abi.ABIPledge.UnpackMethod(param, abi.MethodNameCancelPledge, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, param.Beneficial, param.Amount)
	return nil
}

func (p *MethodCancelPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamCancelPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameCancelPledge, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, sendBlock.AccountAddress, param.Beneficial, NoAgent, NoAgentAddress, NoBid, block.Height)
	if oldPledge == nil || pledgeNotDue(oldPledge, vm) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	if oldPledge.Amount.Sign() != 0 && oldPledge.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	v := util.GetValue(db, beneficialKey)
	oldBeneficial := new(abi.VariablePledgeBeneficial)
	err := abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		util.SetValue(db, pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight, oldPledge.BeneficialAddr, NoAgent, NoAgentAddress, NoBid)
		util.SetValue(db, pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		util.SetValue(db, beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, oldBeneficial.Amount)
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

type MethodAgentPledge struct{}

func (p *MethodAgentPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodAgentPledge) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	param := new(abi.ParamAgentPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentPledge, sendBlock.Data)
	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentPledge, param.PledgeAddress, param.Beneficial, sendBlock.Amount, param.Bid, false)
	return callbackData, true
}

func (p *MethodAgentPledge) GetSendQuota(data []byte) (uint64, error) {
	return AgentPledgeGas, nil
}
func (p *MethodAgentPledge) GetReceiveQuota() uint64 {
	return 0
}

func (p *MethodAgentPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if !util.IsViteToken(block.TokenId) ||
		block.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamAgentPledge)
	if err := abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentPledge, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.StakeHeight < nodeConfig.params.PledgeHeight || param.StakeHeight > PledgeHeightMax {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, param.PledgeAddress, param.Beneficial, param.Bid, param.StakeHeight)
	return nil
}
func (p *MethodAgentPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamAgentPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentPledge, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, param.PledgeAddress, param.Beneficial, Agent, sendBlock.AccountAddress, param.Bid, block.Height)
	var amount *big.Int
	oldWithdrawHeight := uint64(0)
	if oldPledge != nil {
		amount = oldPledge.Amount
		oldWithdrawHeight = oldPledge.WithdrawHeight
	} else {
		amount = big.NewInt(0)
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, amount, helper.Max(oldWithdrawHeight, getPledgeWithdrawHeight(vm, param.StakeHeight)), param.Beneficial, Agent, sendBlock.AccountAddress, param.Bid)
	util.SetValue(db, pledgeKey, pledgeInfo)

	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	oldBeneficialData := util.GetValue(db, beneficialKey)
	var beneficialAmount *big.Int
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(abi.VariablePledgeBeneficial)
		abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	} else {
		beneficialAmount = big.NewInt(0)
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, beneficialAmount)
	util.SetValue(db, beneficialKey, beneficialData)

	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentPledge, param.PledgeAddress, param.Beneficial, sendBlock.Amount, param.Bid, true)
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

type MethodAgentCancelPledge struct{}

func (p *MethodAgentCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodAgentCancelPledge) GetRefundData(sendBlock *ledger.AccountBlock) ([]byte, bool) {
	param := new(abi.ParamAgentCancelPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentCancelPledge, sendBlock.Data)
	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentCancelPledge, param.PledgeAddress, param.Beneficial, param.Amount, param.Bid, false)
	return callbackData, true
}

func (p *MethodAgentCancelPledge) GetSendQuota(data []byte) (uint64, error) {
	return AgentCancelPledgeGas, nil
}

func (p *MethodAgentCancelPledge) GetReceiveQuota() uint64 {
	return 0
}

func (p *MethodAgentCancelPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamAgentCancelPledge)
	if err := abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentCancelPledge, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.Amount.Cmp(pledgeAmountMin) < 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIPledge.PackMethod(abi.MethodNameAgentCancelPledge, param.PledgeAddress, param.Beneficial, param.Amount, param.Bid)
	return nil
}

func (p *MethodAgentCancelPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamAgentCancelPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentCancelPledge, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, param.PledgeAddress, param.Beneficial, Agent, sendBlock.AccountAddress, param.Bid, block.Height)
	if oldPledge == nil || pledgeNotDue(oldPledge, vm) || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	if oldPledge.Amount.Sign() != 0 && oldPledge.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	oldBeneficial := new(abi.VariablePledgeBeneficial)
	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	v := util.GetValue(db, beneficialKey)
	err := abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		util.SetValue(db, pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight, oldPledge.BeneficialAddr, Agent, oldPledge.AgentAddress, param.Bid)
		util.SetValue(db, pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		util.SetValue(db, beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		util.SetValue(db, beneficialKey, pledgeBeneficial)
	}

	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentCancelPledge, param.PledgeAddress, param.Beneficial, param.Amount, param.Bid, true)
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
