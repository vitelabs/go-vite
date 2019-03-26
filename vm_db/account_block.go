package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (db *vmDb) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	return db.chain.GetUnconfirmedBlocks(*db.address)
}
