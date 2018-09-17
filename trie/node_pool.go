package trie

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync"
)

type TrieNodePool struct {
	nodes    map[types.Hash]*TrieNode
	limit    int
	clearNum int

	lock sync.Mutex
}

func NewTrieNodePool() *TrieNodePool {
	return &TrieNodePool{
		nodes:    make(map[types.Hash]*TrieNode),
		limit:    10000000,
		clearNum: 1000000,
	}
}

func (pool *TrieNodePool) Get(key *types.Hash) *TrieNode {
	return pool.nodes[*key]
}

func (pool *TrieNodePool) Set(key *types.Hash, trieNode *TrieNode) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.nodes[*key] = trieNode
	if len(pool.nodes) >= pool.limit {
		pool.clear()
	}
}

func (pool *TrieNodePool) clear() {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	i := 0
	for key := range pool.nodes {
		delete(pool.nodes, key)
		i++
		if i >= pool.clearNum {
			return
		}
	}
}
