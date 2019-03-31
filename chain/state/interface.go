package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type Chain interface {
	IsAccountBlockExisted(hash types.Hash) (bool, error)

	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSnapshotHeightByHash(hash types.Hash) (uint64, error)

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetBalance(addr types.Address, tokenId types.TokenTypeId) (*big.Int, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
}
