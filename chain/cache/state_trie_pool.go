package cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
)

type StateTriePool struct {
	address *types.Address
	trie    *trie.Trie
}

func NewStateTriePool() *StateTriePool {
	return &StateTriePool{}
}

func (*StateTriePool) Set(address *types.Address, trie *trie.Trie) {

}

func (*StateTriePool) Get(address *types.Address) *trie.Trie {
	return nil
}
