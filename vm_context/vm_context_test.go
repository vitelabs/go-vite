package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"log"
	"math/big"
	"testing"
)

var mockSnapshotBlockHash, _ = types.HexToHash("b31fd8301199fc08016060d8a8cc33c6290293c45e66089a2662771f6700bab5")
var mockAccountBlockHash, _ = types.HexToHash("d8c1989526f34211775b2bcdfb3714a4abbe700d9e7b40c064766059a6c38e6c")

var mockTokenTypeId, _ = types.HexToTokenTypeId("tti_000000000000000000004cfd")

var mockAddress, _ = types.HexToAddress("vite_73728b44bda289359fcb298b0d07a6489757a84fb5cfc74527")

func TestVmContext_AddBalance(t *testing.T) {
	vmContext := NewVmContext(&mockSnapshotBlockHash, &mockAccountBlockHash, &mockAddress)
	vmContext.AddBalance(&mockTokenTypeId, big.NewInt(10))
	vmContext.AddBalance(&mockTokenTypeId, big.NewInt(11))
	vmContext.AddBalance(&mockTokenTypeId, big.NewInt(12))
	log.Println(vmContext.ActionList()[0].Params[0].(*types.TokenTypeId).String())
	log.Println(vmContext.ActionList()[0].Params[1].(*big.Int).String())

	log.Println(vmContext.ActionList()[1].Params[0].(*types.TokenTypeId).String())
	log.Println(vmContext.ActionList()[1].Params[1].(*big.Int).String())

	log.Println(vmContext.ActionList()[2].Params[0].(*types.TokenTypeId).String())
	log.Println(vmContext.ActionList()[2].Params[1].(*big.Int).String())
}
