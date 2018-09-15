package vm_context

import "github.com/vitelabs/go-vite/trie"

type StorageIterator struct {
	trieIterator *trie.Iterator
}

func NewStorageIterator(trie *trie.Trie, prefix []byte) *StorageIterator {
	return &StorageIterator{
		trieIterator: trie.NewIterator(prefix),
	}
}

func (si *StorageIterator) Next() (key, value []byte, ok bool) {
	return si.trieIterator.Next()
}
