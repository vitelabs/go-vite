package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	ledger"github.com/vitelabs/go-vite/interfaces/core"
)

func (vdb *vmDb) GetUnconfirmedBlocks(address types.Address) []*ledger.AccountBlock {
	return vdb.chain.GetUnconfirmedBlocks(address)
}
func (vdb *vmDb) GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error) {
	return vdb.chain.GetLatestAccountBlock(addr)
}
