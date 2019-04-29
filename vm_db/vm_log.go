package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (vdb *vmDb) AddLog(log *ledger.VmLog) {
	vdb.unsaved.AddLog(log)
}

func (vdb *vmDb) GetLogList() ledger.VmLogList {
	return vdb.unsaved.GetLogList()
}

func (vdb *vmDb) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return vdb.chain.GetVmLogList(logHash)
}

func (vdb *vmDb) GetLogListHash() *types.Hash {
	return vdb.unsaved.GetLogListHash()
}
