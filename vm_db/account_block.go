package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (db *vmDB) GetUnconfirmedBlocks() ([]*ledger.AccountBlock, error) {
	return db.chain.GetUnconfirmedBlocks(db.address)
}
