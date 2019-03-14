package vm_context

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"github.com/vitelabs/go-vite/vm_db"
)

type VmAccountBlockV2 struct {
	AccountBlock *ledger.AccountBlock
	VmContext    vm_db.VMDB
}

type VmAccountBlock struct {
	AccountBlock *ledger.AccountBlock
	VmContext    vmctxt_interface.VmDatabase
}
