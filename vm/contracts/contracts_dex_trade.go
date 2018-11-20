package contracts

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
	"strings"
)

const (
	VITE = iota
	BTC
	ETH
)

const (
	// dexSuportToken

	jsonDexTrade = `
	[
		{"type":"function","name":"DexTradeNewOrder", "inputs":[{"name":"data","type":"bytes"}]},
		{"type":"function","name":"DexTradeCancelOrder", "inputs":[{"name":"data","type":"bytes"}]}
	]`

	MethodNameDexTradeNewOrder = "DexTradeNewOrder"
	MethodNameDexTradeCancelOrder = "DexTradeCancelOrder"

)


var (
	ABIDexTrade, _ = abi.JSONToABIContract(strings.NewReader(jsonDexFund))
)

type MethodDexTradeNewOrder struct {
}

func (dex *MethodDexTradeNewOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexTradeNewOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex *MethodDexTradeNewOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}

type MethodDexTradeCancelOrder struct {
}

func (dex *MethodDexTradeCancelOrder) GetFee(context contractsContext, block *vm_context.VmAccountBlock) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (dex *MethodDexTradeCancelOrder) DoSend(context contractsContext, block *vm_context.VmAccountBlock, quotaLeft uint64) (uint64, error) {
	return 0, nil
}

func (dex MethodDexTradeCancelOrder) DoReceive(context contractsContext, block *vm_context.VmAccountBlock, sendBlock *ledger.AccountBlock) error {
	return nil
}