package trie

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
)

type Trie struct {
	db *leveldb.DB

	RootHash *types.Hash
	Root     *TrieNode
}

func NewTrie(db *leveldb.DB, rootHash *types.Hash) (*Trie, error) {
	trie := &Trie{
		db:       db,
		RootHash: rootHash,
	}

	trie.loadCacheFromDb()
	return trie, nil
}

func (trie *Trie) loadCacheFromDb() error {
	return nil
}

func (trie *Trie) computeHash() {

}

func (trie *Trie) Copy() *Trie {
	return &Trie{
		Root: trie.Root.Copy(),
	}
}

func (trie *Trie) Save() {

}

func (trie *Trie) SetValue(key []byte, value []byte) {

}

func (trie *Trie) GetValue(key []byte) []byte {
	return nil
}
