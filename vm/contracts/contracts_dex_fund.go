package contracts

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strings"
)

const (
	jsonDexFund = `
	[
		{"type":"function","name":"DexFundUserTopUp", "inputs":[{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"DexFundUserWithdraw","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundSystemFreezeBalance","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundUserShareTradeFee","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexFundSystemFillOrder","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},	
		{"type":"function","name":"DexFundUnfreezeBalance","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
        {"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`

	MethodNameDexFundUserTopUp           = "DexFundUserTopUp "
	MethodNameDexFundUserWithdraw        = "DexFundUserWithdraw"
	MethodNameDexFundSystemFreezeBalance = "DexFundSystemFreezeBalance"
	MethodNameDexFundUserShareTradeFee   = "DexFundUserShareTradeFee"
	MethodNameDexFundSystemFillOrder     = "DexFundSystemFillOrder"
	MethodNameDexFundUnfreezeBalance     = "DexFundUnfreezeBalance"

	//VariableNamePledgeInfo       = "pledgeInfo"
	//VariableNamePledgeBeneficial = "pledgeBeneficial"
)

var (
	ABIDexFund, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
)

type MethodDexFundUserTopUp struct {
}

func (dex *MethodDexFundUserTopUp) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexFundUserTopUp) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexFundUserTopUp) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexFundUserWithdraw struct {
}

func (dex *MethodDexFundUserWithdraw) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexFundUserWithdraw) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexFundUserWithdraw) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexFundSystemFreezeBalance struct {
}

func (dex *MethodDexFundSystemFreezeBalance) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexFundSystemFreezeBalance) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexFundSystemFreezeBalance) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexFundUserShareTradeFee struct {
}

func (dex *MethodDexFundUserShareTradeFee) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexFundUserShareTradeFee) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexFundUserShareTradeFee) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexFundSystemFillOrder struct {
}

func (dex *MethodDexFundSystemFillOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexFundSystemFillOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexFundSystemFillOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexFundUnfreezeBalance struct {
}

func (dex *MethodDexFundUnfreezeBalance) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexFundUnfreezeBalance) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexFundUnfreezeBalance) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}
