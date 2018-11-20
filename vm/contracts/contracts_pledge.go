package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

type MethodPledge struct{}

func (p *MethodPledge) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodPledge) GetRefundData() []byte {
	return []byte{1}
}

// pledge ViteToken for a beneficial to get quota
func (p *MethodPledge) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	// pledge gas is low without data gas cost, so that a new account is easy to pledge
	quotaLeft, err := util.UseQuota(quotaLeft, PledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Cmp(pledgeAmountMin) < 0 ||
		!util.IsViteToken(block.TokenId) ||
		!IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	beneficialAddr := new(types.Address)
	if err = cabi.ABIPledge.UnpackMethod(beneficialAddr, cabi.MethodNamePledge, block.Data); err != nil {
		return quotaLeft, errors.New("invalid beneficial address")
	}
	return quotaLeft, nil
}
func (p *MethodPledge) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	beneficialAddr := new(types.Address)
	cabi.ABIPledge.UnpackMethod(beneficialAddr, cabi.MethodNamePledge, sendBlock.Data)
	beneficialKey := cabi.GetPledgeBeneficialKey(*beneficialAddr)
	pledgeKey := cabi.GetPledgeKey(sendBlock.AccountAddress, beneficialKey)
	oldPledgeData := db.GetStorage(&block.AccountAddress, pledgeKey)
	amount := big.NewInt(0)
	if len(oldPledgeData) > 0 {
		oldPledge := new(cabi.PledgeInfo)
		cabi.ABIPledge.UnpackVariable(oldPledge, cabi.VariableNamePledgeInfo, oldPledgeData)
		amount = oldPledge.Amount
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := cabi.ABIPledge.PackVariable(cabi.VariableNamePledgeInfo, amount, db.CurrentSnapshotBlock().Height+nodeConfig.params.MinPledgeHeight)
	db.SetStorage(pledgeKey, pledgeInfo)

	oldBeneficialData := db.GetStorage(&block.AccountAddress, beneficialKey)
	beneficialAmount := big.NewInt(0)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(cabi.VariablePledgeBeneficial)
		cabi.ABIPledge.UnpackVariable(oldBeneficial, cabi.VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := cabi.ABIPledge.PackVariable(cabi.VariableNamePledgeBeneficial, beneficialAmount)
	db.SetStorage(beneficialKey, beneficialData)
	return nil, nil
}

type MethodCancelPledge struct{}

func (p *MethodCancelPledge) GetFee(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (p *MethodCancelPledge) GetRefundData() []byte {
	return []byte{2}
}

// cancel pledge ViteToken
func (p *MethodCancelPledge) DoSend(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.Amount.Sign() > 0 ||
		!IsUserAccount(db, block.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(cabi.ParamCancelPledge)
	if err = cabi.ABIPledge.UnpackMethod(param, cabi.MethodNameCancelPledge, block.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	if param.Amount.Sign() == 0 {
		return quotaLeft, errors.New("cancel pledge amount is 0")
	}
	return quotaLeft, nil
}

func (p *MethodCancelPledge) DoReceive(db vmctxt_interface.VmDatabase, block *ledger.AccountBlock, sendBlock *ledger.AccountBlock) ([]*SendBlock, error) {
	param := new(cabi.ParamCancelPledge)
	cabi.ABIPledge.UnpackMethod(param, cabi.MethodNameCancelPledge, sendBlock.Data)
	beneficialKey := cabi.GetPledgeBeneficialKey(param.Beneficial)
	pledgeKey := cabi.GetPledgeKey(sendBlock.AccountAddress, beneficialKey)
	oldPledge := new(cabi.PledgeInfo)
	err := cabi.ABIPledge.UnpackVariable(oldPledge, cabi.VariableNamePledgeInfo, db.GetStorage(&block.AccountAddress, pledgeKey))
	if err != nil || oldPledge.WithdrawHeight > db.CurrentSnapshotBlock().Height || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return nil, errors.New("pledge not yet due")
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(cabi.VariablePledgeBeneficial)
	err = cabi.ABIPledge.UnpackVariable(oldBeneficial, cabi.VariableNamePledgeBeneficial, db.GetStorage(&block.AccountAddress, beneficialKey))
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return nil, errors.New("invalid pledge amount")
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		db.SetStorage(pledgeKey, nil)
	} else {
		pledgeInfo, _ := cabi.ABIPledge.PackVariable(cabi.VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight)
		db.SetStorage(pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		db.SetStorage(beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := cabi.ABIPledge.PackVariable(cabi.VariableNamePledgeBeneficial, oldBeneficial.Amount)
		db.SetStorage(beneficialKey, pledgeBeneficial)
	}
	return []*SendBlock{
		{
			block,
			sendBlock.AccountAddress,
			ledger.BlockTypeSendCall,
			param.Amount,
			ledger.ViteTokenId,
			[]byte{},
		},
	}, nil
}
