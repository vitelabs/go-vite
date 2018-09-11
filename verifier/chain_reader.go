package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotReader interface {
	GenesisSnapshot() *ledger.SnapshotBlock
	HeadSnapshot() (*ledger.SnapshotBlock, error)
	GetSnapshotByHash(hash *types.Hash) *ledger.SnapshotBlock
}
type AccountReader interface {
	HeadAccount(address *types.Address) (*ledger.AccountBlock, error)
	GetAccountByHash(address *types.Address, hash *types.Hash) *ledger.AccountBlock

	GetAccountByFromHash(address *types.Address, fromHash *types.Hash) *ledger.AccountBlock
}

type ChainReader interface {
	AccountReader
	SnapshotReader
}
