package contracts

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strings"
)

const (
	jsonDexTrade = `
	[
		{"type":"function","name":"DexTradeUserNewOrder", "inputs":[{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"DexTradeSystemPendingOrder","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexTradeUserStopOrder","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"function","name":"DexTradeSystemStopOrder","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
        {"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
	]`

	MethodNameDexTradeUserNewOrder       = "DexTradeUserNewOrder"
	MethodNameDexTradeSystemPendingOrder = "DexTradeSystemPendingOrder"
	MethodNameDexTradeUserStopOrder      = "DexTradeUserStopOrder"
	MethodNameDexTradeSystemStopOrder    = "DexTradeSystemStopOrder"

	//VariableNamePledgeInfo       = "pledgeInfo"
	//VariableNamePledgeBeneficial = "pledgeBeneficial"
)

var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexTrade))
)

type MethodDexTradeUserNewOrder struct {
}

func (dex *MethodDexTradeUserNewOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexTradeUserNewOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexTradeUserNewOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexTradeSystemPendingOrder struct {
}

func (dex *MethodDexTradeSystemPendingOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexTradeSystemPendingOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexTradeSystemPendingOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexTradeUserStopOrder struct {
}

func (dex *MethodDexTradeUserStopOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexTradeUserStopOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexTradeUserStopOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexTradeSystemStopOrder struct {
}

func (dex *MethodDexTradeSystemStopOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexTradeSystemStopOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexTradeSystemStopOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}
