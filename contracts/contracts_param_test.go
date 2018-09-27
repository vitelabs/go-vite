package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestPackMethodParam(t *testing.T) {
	addr, _, _ := types.CreateAddress()
	_, err := PackMethodParam(AddressRegister, MethodNameRegister, types.DELEGATE_GID, "node", addr, addr)
	if err != nil {
		t.Fatalf("pack method param failed")
	}
}
