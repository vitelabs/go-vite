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

func (p *MethodPledge) GetRefundData() ([]byte, bool) {
	return []byte{1}, false
}

func (p *MethodPledge) GetSendQuota(data []byte) (uint64, error) {
	return PledgeGas, nil
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
	amount := big.NewInt(0)
	if oldPledge != nil {
		amount = oldPledge.Amount
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, amount, vm.GlobalStatus().SnapshotBlock().Height+nodeConfig.params.PledgeHeight, beneficialAddr, NoAgent, NoAgentAddress, NoBid)
	db.SetValue(pledgeKey, pledgeInfo)

	beneficialKey := abi.GetPledgeBeneficialKey(*beneficialAddr)
	oldBeneficialData, err := db.GetValue(beneficialKey)
	if err != nil {
		return nil, err
	}
	beneficialAmount := big.NewInt(0)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(abi.VariablePledgeBeneficial)
		abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, beneficialAmount)
	db.SetValue(beneficialKey, beneficialData)
	return nil, nil
}

func getPledgeInfo(db vm_db.VmDb, pledgeAddr types.Address, beneficialAddr types.Address, agent bool, agentAddr types.Address, bid uint8, currentIndex uint64) ([]byte, *abi.PledgeInfo) {
	iterator, err := db.NewStorageIterator(abi.GetPledgeKeyPrefix(pledgeAddr))
	util.DealWithErr(err)
	defer iterator.Release()
	maxUint64 := uint64(0)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				util.DealWithErr(err)
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
		maxUint64 = helper.Max(maxUint64, abi.GetIndexFromPledgeKey(iterator.Key()))
	}
	if maxUint64 < currentIndex {
		return abi.GetPledgeKey(pledgeAddr, currentIndex), nil
	} else {
		return abi.GetPledgeKey(pledgeAddr, maxUint64+1), nil
	}
}

type MethodCancelPledge struct{}

func (p *MethodCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelPledge) GetRefundData() ([]byte, bool) {
	return []byte{2}, false
}

func (p *MethodCancelPledge) GetSendQuota(data []byte) (uint64, error) {
	return CancelPledgeGas, nil
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
	if oldPledge == nil || oldPledge.WithdrawHeight > vm.GlobalStatus().SnapshotBlock().Height || oldPledge.Amount.Cmp(param.Amount) < 0 || oldPledge.BeneficialAddr != param.Beneficial {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	if oldPledge.Amount.Sign() != 0 && oldPledge.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	oldBeneficial := new(abi.VariablePledgeBeneficial)
	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	v, err := db.GetValue(beneficialKey)
	util.DealWithErr(err)
	err = abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		db.SetValue(pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight, oldPledge.BeneficialAddr, NoAgent, NoAgentAddress, NoBid)
		db.SetValue(pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		db.SetValue(beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		db.SetValue(beneficialKey, pledgeBeneficial)
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

func (p *MethodAgentPledge) GetRefundData() ([]byte, bool) {
	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentPledge, false)
	return callbackData, true
}

func (p *MethodAgentPledge) GetSendQuota(data []byte) (uint64, error) {
	return AgentPledgeGas, nil
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
	block.Data, _ = abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, param.PledgeAddress, param.Beneficial, param.Bid)
	return nil
}
func (p *MethodAgentPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, vm vmEnvironment) ([]*ledger.AccountBlock, error) {
	param := new(abi.ParamAgentPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameAgentPledge, sendBlock.Data)
	pledgeKey, oldPledge := getPledgeInfo(db, param.PledgeAddress, param.Beneficial, Agent, sendBlock.AccountAddress, param.Bid, block.Height)
	amount := big.NewInt(0)
	if oldPledge != nil {
		amount = oldPledge.Amount
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, amount, vm.GlobalStatus().SnapshotBlock().Height+nodeConfig.params.PledgeHeight, param.Beneficial, Agent, sendBlock.AccountAddress, param.Bid)
	db.SetValue(pledgeKey, pledgeInfo)

	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	oldBeneficialData, err := db.GetValue(beneficialKey)
	if err != nil {
		return nil, err
	}
	beneficialAmount := big.NewInt(0)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(abi.VariablePledgeBeneficial)
		abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, beneficialAmount)
	db.SetValue(beneficialKey, beneficialData)

	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentPledge, true)
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

func (p *MethodAgentCancelPledge) GetRefundData() ([]byte, bool) {
	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentCancelPledge, false)
	return callbackData, true
}

func (p *MethodAgentCancelPledge) GetSendQuota(data []byte) (uint64, error) {
	return AgentCancelPledgeGas, nil
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
	if oldPledge == nil || oldPledge.WithdrawHeight > vm.GlobalStatus().SnapshotBlock().Height || oldPledge.Amount.Cmp(param.Amount) < 0 || oldPledge.BeneficialAddr != param.Beneficial {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	if oldPledge.Amount.Sign() != 0 && oldPledge.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	oldBeneficial := new(abi.VariablePledgeBeneficial)
	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	v, err := db.GetValue(beneficialKey)
	util.DealWithErr(err)
	err = abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		db.SetValue(pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight, oldPledge.BeneficialAddr, Agent, oldPledge.AgentAddress, param.Bid)
		db.SetValue(pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		db.SetValue(beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		db.SetValue(beneficialKey, pledgeBeneficial)
	}

	callbackData, _ := abi.ABIPledge.PackCallback(abi.MethodNameAgentCancelPledge, true)
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
