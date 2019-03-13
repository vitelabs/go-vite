package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDb) AddLog(log *ledger.VmLog) {
	db.unsaved.AddLog(log)
}

func (db *vmDb) GetLogList(logHash *types.Hash) ledger.VmLogList {
	return db.unsaved.GetLogList()
}

func (db *vmDb) GetHistoryLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return db.chain.GetLogList(logHash)
}

func (db *vmDb) GetLogListHash() *types.Hash {
	return db.unsaved.GetLogListHash()
}
