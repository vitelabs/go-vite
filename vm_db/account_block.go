package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (vdb *vmDb) GetUnconfirmedBlocks() []*ledger.AccountBlock {
	return vdb.chain.GetUnconfirmedBlocks(*vdb.address)
}
func (vdb *vmDb) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return vdb.chain.GetAccountBlockByHash(blockHash)
}
func (vdb *vmDb) GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return vdb.chain.GetCompleteBlockByHash(blockHash)
}
