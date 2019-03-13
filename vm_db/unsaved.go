package vm_db

import (
	"github.com/vitelabs/go-vite/ledger"
)

type Unsaved struct {
	//contractGidList []vmctxt_interface.ContractGid

	logList ledger.VmLogList
	storage map[string][]byte
}

func NewUnsaved() *Unsaved {
	return &Unsaved{}
}
