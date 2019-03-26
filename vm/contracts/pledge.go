package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

type MethodPledge struct{}

func (p *MethodPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodPledge) GetRefundData() []byte {
	return []byte{1}
}

func (p *MethodPledge) GetSendQuota(data []byte) (uint64, error) {
	return PledgeGas, nil
}

// pledge ViteToken for a beneficial to get quota
func (p *MethodPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	// pledge gas is low without data gas cost, so that a new account is easy to pledge
	if !util.IsViteToken(block.TokenId) ||
		!util.IsUserAccount(db) ||
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
func (p *MethodPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	beneficialAddr := new(types.Address)
	abi.ABIPledge.UnpackMethod(beneficialAddr, abi.MethodNamePledge, sendBlock.Data)
	pledgeKey := abi.GetPledgeKey(sendBlock.AccountAddress, *beneficialAddr)
	oldPledgeData, err := db.GetValue(pledgeKey)
	if err != nil {
		return nil, err
	}
	amount := big.NewInt(0)
	if len(oldPledgeData) > 0 {
		oldPledge := new(abi.PledgeInfo)
		abi.ABIPledge.UnpackVariable(oldPledge, abi.VariableNamePledgeInfo, oldPledgeData)
		amount = oldPledge.Amount
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, amount, globalStatus.SnapshotBlock.Height+nodeConfig.params.PledgeHeight)
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

type MethodCancelPledge struct{}

func (p *MethodCancelPledge) GetFee(block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelPledge) GetRefundData() []byte {
	return []byte{2}
}

func (p *MethodCancelPledge) GetSendQuota(data []byte) (uint64, error) {
	return CancelPledgeGas, nil
}

// cancel pledge ViteToken
func (p *MethodCancelPledge) DoSend(db vm_db.VmDb, block *ledger.AccountBlock) error {
	if block.Amount.Sign() > 0 ||
		!util.IsUserAccount(db) {
		return util.ErrInvalidMethodParam
	}
	param := new(abi.ParamCancelPledge)
	if err := abi.ABIPledge.UnpackMethod(param, abi.MethodNameCancelPledge, block.Data); err != nil {
		return util.ErrInvalidMethodParam
	}
	if param.Amount.Sign() == 0 {
		return util.ErrInvalidMethodParam
	}
	block.Data, _ = abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, param.Beneficial, param.Amount)
	return nil
}

func (p *MethodCancelPledge) DoReceive(db vm_db.VmDb, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock, globalStatus *util.GlobalStatus) ([]*SendBlock, error) {
	param := new(abi.ParamCancelPledge)
	abi.ABIPledge.UnpackMethod(param, abi.MethodNameCancelPledge, sendBlock.Data)
	pledgeKey := abi.GetPledgeKey(sendBlock.AccountAddress, param.Beneficial)
	oldPledge := new(abi.PledgeInfo)
	v, err := db.GetValue(pledgeKey)
	util.DealWithErr(err)
	err = abi.ABIPledge.UnpackVariable(oldPledge, abi.VariableNamePledgeInfo, v)
	if err != nil || oldPledge.WithdrawHeight > globalStatus.SnapshotBlock.Height || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)

	oldBeneficial := new(abi.VariablePledgeBeneficial)
	beneficialKey := abi.GetPledgeBeneficialKey(param.Beneficial)
	v, err = db.GetValue(beneficialKey)
	util.DealWithErr(err)
	err = abi.ABIPledge.UnpackVariable(oldBeneficial, abi.VariableNamePledgeBeneficial, v)
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, util.ErrInvalidMethodParam
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)
	if oldBeneficial.Amount.Sign() != 0 && oldBeneficial.Amount.Cmp(pledgeAmountMin) < 0 {
		return nil, util.ErrInvalidMethodParam
	}

	if oldPledge.Amount.Sign() == 0 {
		db.SetValue(pledgeKey, nil)
	} else {
		pledgeInfo, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight)
		db.SetValue(pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		db.SetValue(beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := abi.ABIPledge.PackVariable(abi.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		db.SetValue(beneficialKey, pledgeBeneficial)
	}
	return []*SendBlock{
		{
			sendBlock.AccountAddress,
			ledger.BlockTypeSendCall,
			param.Amount,
			ledger.ViteTokenId,
			[]byte{},
		},
	}, nil
}
