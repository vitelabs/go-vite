package contracts

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strings"
)

const (
	jsonPledge = `
	[
		{"type":"function","name":"Pledge", "inputs":[{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"CancelPledge","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`

	MethodNamePledge             = "Pledge"
	MethodNameCancelPledge       = "CancelPledge"
	VariableNamePledgeInfo       = "pledgeInfo"
	VariableNamePledgeBeneficial = "pledgeBeneficial"
)

var (
	ABIPledge, _ = abi.JSONToABIContract(strings.NewReader(jsonPledge))
)

type VariablePledgeBeneficial struct {
	Amount *big.Int
}
type ParamCancelPledge struct {
	Beneficial types.Address
	Amount     *big.Int
}
type PledgeInfo struct {
	Amount         *big.Int
	WithdrawHeight uint64
	BeneficialAddr types.Address
}

func GetPledgeBeneficialKey(beneficial types.Address) []byte {
	return beneficial.Bytes()
}
func GetPledgeKey(addr types.Address, pledgeBeneficialKey []byte) []byte {
	return append(addr.Bytes(), pledgeBeneficialKey...)
}
func IsPledgeKey(key []byte) bool {
	return len(key) > types.AddressSize
}
func GetBeneficialFromPledgeKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[types.AddressSize:])
	return address
}

func GetPledgeBeneficialAmount(db StorageDatabase, beneficial types.Address) *big.Int {
	key := GetPledgeBeneficialKey(beneficial)
	beneficialAmount := new(VariablePledgeBeneficial)
	err := ABIPledge.UnpackVariable(beneficialAmount, VariableNamePledgeBeneficial, db.GetStorage(&AddressPledge, key))
	if err == nil {
		return beneficialAmount.Amount
	}
	return big.NewInt(0)
}

func GetPledgeInfoList(db StorageDatabase, addr types.Address) ([]*PledgeInfo, *big.Int) {
	pledgeAmount := big.NewInt(0)
	iterator := db.NewStorageIterator(&AddressPledge, addr.Bytes())
	pledgeInfoList := make([]*PledgeInfo, 0)
	if iterator == nil {
		return pledgeInfoList, pledgeAmount
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if IsPledgeKey(key) {
			pledgeInfo := new(PledgeInfo)
			if err := ABIPledge.UnpackVariable(pledgeInfo, VariableNamePledgeInfo, value); err == nil {
				pledgeInfo.BeneficialAddr = GetBeneficialFromPledgeKey(key)
				pledgeInfoList = append(pledgeInfoList, pledgeInfo)
				pledgeAmount.Add(pledgeAmount, pledgeInfo.Amount)
			}
		}
	}
	return pledgeInfoList, pledgeAmount
}

type MethodPledge struct{}

func (p *MethodPledge) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// pledge ViteToken for a beneficial to get quota
func (p *MethodPledge) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	// pledge gas is low without data gas cost, so that a new account is easy to pledge
	quotaLeft, err := util.UseQuota(quotaLeft, PledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Cmp(pledgeAmountMin) < 0 ||
		!util.IsViteToken(block.AccountBlock.TokenId) ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	beneficialAddr := new(types.Address)
	if err = ABIPledge.UnpackMethod(beneficialAddr, MethodNamePledge, block.AccountBlock.Data); err != nil {
		return quotaLeft, errors.New("invalid beneficial address")
	}
	return quotaLeft, nil
}
func (p *MethodPledge) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	beneficialAddr := new(types.Address)
	ABIPledge.UnpackMethod(beneficialAddr, MethodNamePledge, sendBlock.Data)
	beneficialKey := GetPledgeBeneficialKey(*beneficialAddr)
	pledgeKey := GetPledgeKey(sendBlock.AccountAddress, beneficialKey)
	oldPledgeData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, pledgeKey)
	amount := new(big.Int)
	if len(oldPledgeData) > 0 {
		oldPledge := new(PledgeInfo)
		ABIPledge.UnpackVariable(oldPledge, VariableNamePledgeInfo, oldPledgeData)
		amount = oldPledge.Amount
	}
	amount.Add(amount, sendBlock.Amount)
	pledgeInfo, _ := ABIPledge.PackVariable(VariableNamePledgeInfo, amount, block.VmContext.CurrentSnapshotBlock().Height+nodeConfig.params.MinPledgeHeight)
	block.VmContext.SetStorage(pledgeKey, pledgeInfo)

	oldBeneficialData := block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, beneficialKey)
	beneficialAmount := new(big.Int)
	if len(oldBeneficialData) > 0 {
		oldBeneficial := new(VariablePledgeBeneficial)
		ABIPledge.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, oldBeneficialData)
		beneficialAmount = oldBeneficial.Amount
	}
	beneficialAmount.Add(beneficialAmount, sendBlock.Amount)
	beneficialData, _ := ABIPledge.PackVariable(VariableNamePledgeBeneficial, beneficialAmount)
	block.VmContext.SetStorage(beneficialKey, beneficialData)
	return nil
}

type MethodCancelPledge struct{}

func (p *MethodCancelPledge) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

// cancel pledge ViteToken
func (p *MethodCancelPledge) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	quotaLeft, err := util.UseQuota(quotaLeft, CancelPledgeGas)
	if err != nil {
		return quotaLeft, err
	}
	if block.AccountBlock.Amount.Sign() > 0 ||
		!IsUserAccount(block.VmContext, block.AccountBlock.AccountAddress) {
		return quotaLeft, errors.New("invalid block data")
	}
	param := new(ParamCancelPledge)
	if err = ABIPledge.UnpackMethod(param, MethodNameCancelPledge, block.AccountBlock.Data); err != nil {
		return quotaLeft, util.ErrInvalidMethodParam
	}
	return quotaLeft, nil
}

func (p *MethodCancelPledge) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	param := new(ParamCancelPledge)
	ABIPledge.UnpackMethod(param, MethodNameCancelPledge, sendBlock.Data)
	beneficialKey := GetPledgeBeneficialKey(param.Beneficial)
	pledgeKey := GetPledgeKey(sendBlock.AccountAddress, beneficialKey)
	oldPledge := new(PledgeInfo)
	err := ABIPledge.UnpackVariable(oldPledge, VariableNamePledgeInfo, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, pledgeKey))
	if err != nil || oldPledge.WithdrawHeight > block.VmContext.CurrentSnapshotBlock().Height || oldPledge.Amount.Cmp(param.Amount) < 0 {
		return errors.New("pledge not yet due")
	}
	oldPledge.Amount.Sub(oldPledge.Amount, param.Amount)
	oldBeneficial := new(VariablePledgeBeneficial)
	err = ABIPledge.UnpackVariable(oldBeneficial, VariableNamePledgeBeneficial, block.VmContext.GetStorage(&block.AccountBlock.AccountAddress, beneficialKey))
	if err != nil || oldBeneficial.Amount.Cmp(param.Amount) < 0 {
		return errors.New("invalid pledge amount")
	}
	oldBeneficial.Amount.Sub(oldBeneficial.Amount, param.Amount)

	if oldPledge.Amount.Sign() == 0 {
		block.VmContext.SetStorage(pledgeKey, nil)
	} else {
		pledgeInfo, _ := ABIPledge.PackVariable(VariableNamePledgeInfo, oldPledge.Amount, oldPledge.WithdrawHeight)
		block.VmContext.SetStorage(pledgeKey, pledgeInfo)
	}

	if oldBeneficial.Amount.Sign() == 0 {
		block.VmContext.SetStorage(beneficialKey, nil)
	} else {
		pledgeBeneficial, _ := ABIPledge.PackVariable(VariableNamePledgeBeneficial, oldBeneficial.Amount)
		block.VmContext.SetStorage(beneficialKey, pledgeBeneficial)
	}

	context.AppendBlock(
		&vm_context.VmAccountBlock{
			util.MakeSendBlock(
				block.AccountBlock,
				sendBlock.AccountAddress,
				ledger.BlockTypeSendCall,
				param.Amount,
				ledger.ViteTokenId,
				context.GetNewBlockHeight(block),
				[]byte{}),
			nil})
	return nil
}
