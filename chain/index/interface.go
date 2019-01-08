package chain_index

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetLatestBlockEventId() (uint64, error)

	GetEvent(eventId uint64) (byte, []types.Hash, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)

	GetAccount(address *types.Address) (*ledger.Account, error)
	IsAccountBlockExisted(hash types.Hash) (bool, error)
	ChainDb() *chain_db.ChainDb
	IsGenesisAccountBlock(block *ledger.AccountBlock) bool
}
