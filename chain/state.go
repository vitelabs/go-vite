package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
)

const (
	SAVE_TRIE_STATUS_STOPPED = 1
	SAVE_TRIE_STATUS_STARTED = 2
)

func (c *chain) CleanTrieNodePool() {
	c.trieNodePool.Clear()
}

func (c *chain) StopSaveTrie() {
	c.saveTrieStatusLock.Lock()
	defer c.saveTrieStatusLock.Unlock()
	if c.saveTrieStatus == SAVE_TRIE_STATUS_STOPPED {
		return
	}

	c.saveTrieLock.Lock()
	c.saveTrieStatus = SAVE_TRIE_STATUS_STOPPED
}
func (c *chain) StartSaveTrie() {
	c.saveTrieStatusLock.Lock()
	defer c.saveTrieStatusLock.Unlock()
	if c.saveTrieStatus == SAVE_TRIE_STATUS_STARTED {
		return
	}

	c.saveTrieLock.Unlock()
	c.saveTrieStatus = SAVE_TRIE_STATUS_STARTED
}

func (c *chain) ShallowCheckStateTrie(stateHash *types.Hash) (bool, error) {
	return trie.ShallowCheck(c.TrieDb(), stateHash)
}

func (c *chain) GetStateTrie(stateHash *types.Hash) *trie.Trie {
	return trie.NewTrie(c.TrieDb(), stateHash, c.trieNodePool)
}

func (c *chain) NewStateTrie() *trie.Trie {
	return trie.NewTrie(c.TrieDb(), nil, c.trieNodePool)
}
