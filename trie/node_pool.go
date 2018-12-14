package trie

import (
	"sync"

	"github.com/vitelabs/go-vite/common/types"
)

type TrieNodePool struct {
	nodes    map[types.Hash]*TrieNode
	limit    int
	clearNum int

	lock sync.RWMutex
}

func NewCustomTrieNodePool(limit int, clearNum int) *TrieNodePool {
	return &TrieNodePool{
		nodes:    make(map[types.Hash]*TrieNode),
		limit:    limit,
		clearNum: clearNum,
	}
}

func NewTrieNodePool() *TrieNodePool {
	return &TrieNodePool{
		nodes:    make(map[types.Hash]*TrieNode),
		limit:    100000,
		clearNum: 50000,
	}
}

func (pool *TrieNodePool) Get(key *types.Hash) *TrieNode {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

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

func (pool *TrieNodePool) Clear() {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	pool.nodes = make(map[types.Hash]*TrieNode)
}

func (pool *TrieNodePool) clear() {
	i := 0
	for key := range pool.nodes {
		delete(pool.nodes, key)
		i++
		if i >= pool.clearNum {
			return
		}
	}
}
