package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
)

func (c *chain) GetStateTrie(stateHash *types.Hash) *trie.Trie {
	return trie.NewTrie(c.chainDb.Db(), stateHash, c.trieNodePool)
}

func (c *chain) NewStateTrie() *trie.Trie {
	return trie.NewTrie(c.chainDb.Db(), nil, c.trieNodePool)
}
