package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (vdb *vmDb) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	return vdb.chain.GetUnconfirmedBlocks(*vdb.address)
}
