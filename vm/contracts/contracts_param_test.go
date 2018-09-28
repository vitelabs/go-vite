package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func TestPackMethodParam(t *testing.T) {
	addr, _, _ := types.CreateAddress()
	_, err := PackMethodParam(AddressRegister, MethodNameRegister, types.DELEGATE_GID, "node", addr, addr)
	if err != nil {
		t.Fatalf("pack method param failed")
	}
}

func TestPackConsensusGroupConditionParam(t *testing.T) {
	_, err := PackConsensusGroupConditionParam(RegisterConditionPrefix, uint8(1), big.NewInt(1), ledger.ViteTokenId, int64(10))
	if err != nil {
		t.Fatalf("pack consensus group condition param failed")
	}
}
