package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetAccount(address *types.Address) *ledger.Account
	GetSnapshotBlockByHash(hash *types.Hash) (block *ledger.SnapshotBlock, returnErr error)
	GetAccountBlockByHash(blockHash *types.Hash) (block *ledger.AccountBlock, returnErr error)
}
