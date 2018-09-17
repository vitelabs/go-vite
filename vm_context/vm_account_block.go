package vm_context

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type VmAccountBlock struct {
	AccountBlock *ledger.AccountBlock
	VmContext    vmctxt_interface.VmDatabase
}
