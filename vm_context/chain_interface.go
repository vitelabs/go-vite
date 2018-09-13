package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
)

type Chain interface {
	GetAccount(address *types.Address) *ledger.Account
	GetSnapshotBlockByHash(hash *types.Hash) (block *ledger.SnapshotBlock, returnErr error)
	GetAccountBlockByHash(blockHash *types.Hash) (block *ledger.AccountBlock, returnErr error)
	GetStateTrie(hash *types.Hash) *trie.Trie
	GetTokenInfoById(tokenId *types.TokenTypeId) (*ledger.Token, error)

	NewStateTrie() *trie.Trie
}
