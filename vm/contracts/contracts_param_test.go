package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"testing"
)

func TestPackMethodParam(t *testing.T) {
	_, err := PackMethodParam(AddressVote, MethodNameVote, types.DELEGATE_GID, "node")
	if err != nil {
		t.Fatalf("pack method param failed, %v", err)
	}
}

func TestPackConsensusGroupConditionParam(t *testing.T) {
	_, err := PackConsensusGroupConditionParam(RegisterConditionPrefix, uint8(1), big.NewInt(1), ledger.ViteTokenId, uint64(10))
	if err != nil {
		t.Fatalf("pack consensus group condition param failed")
	}
}
