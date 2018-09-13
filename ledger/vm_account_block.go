package ledger

import "github.com/vitelabs/go-vite/vm_context"

type VmAccountBlock struct {
	AccountBlock *AccountBlock
	VmContext    *vm_context.VmContext
}
