package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
	"sync"
)

type StateTriePool struct {
	cache   map[types.Address]*trie.Trie
	chain   *Chain
	setLock sync.Mutex
}

func NewStateTriePool(chain *Chain) *StateTriePool {
	return &StateTriePool{
		cache: make(map[types.Address]*trie.Trie),
		chain: chain,
	}
}

func (pool *StateTriePool) unsafeSet(address *types.Address, trie *trie.Trie) {
	pool.cache[*address] = trie
}

func (pool *StateTriePool) Set(address *types.Address, trie *trie.Trie) {
	pool.setLock.Lock()
	defer pool.setLock.Unlock()
	pool.unsafeSet(address, trie)
}

func (pool *StateTriePool) Get(address *types.Address) (*trie.Trie, error) {
	if cachedTrie := pool.cache[*address]; cachedTrie != nil {
		return cachedTrie, nil
	}

	pool.setLock.Lock()
	defer pool.setLock.Unlock()

	latestBlock, err := pool.chain.GetLatestAccountBlock(address)
	if err != nil {
		return nil, err
	}

	if latestBlock != nil {
		stateTir := pool.chain.GetStateTrie(&latestBlock.StateHash)
		pool.unsafeSet(address, stateTir)
		return stateTir, nil
	}
	return nil, nil
}
