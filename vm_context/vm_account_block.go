package vm_context

import "github.com/vitelabs/go-vite/ledger"

type VmAccountBlock struct {
	AccountBlock *ledger.AccountBlock
	VmContext    *VmContext
}
