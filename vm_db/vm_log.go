package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDB) AddLog(log *ledger.VmLog) {
	db.unsaved.AddLog(log)
}

func (db *vmDB) GetLogList(logHash *types.Hash) (ledger.VmLogList, error) {
	return db.chain.GetLogList(logHash)
}
func (db *vmDB) GetLogListHash() *types.Hash {
	return db.unsaved.GetLogListHash()
}
