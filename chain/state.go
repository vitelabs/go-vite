package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
)

func (c *Chain) GetStateTrie(stateHash *types.Hash) *trie.Trie {
	return trie.NewTrie(c.chainDb.Db(), stateHash, c.trieNodePool)
}

func (c *Chain) NewStateTrie() *trie.Trie {
	return trie.NewTrie(c.chainDb.Db(), nil, c.trieNodePool)
}
