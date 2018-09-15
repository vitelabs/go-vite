package trie

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common"
	"path/filepath"
	"testing"
)

func TestNewIterator(t *testing.T) {
	db := database.NewLevelDb(filepath.Join(common.GoViteTestDataDir(), "trie"))
	defer db.Close()

	pool := NewTrieNodePool()

	trie := NewTrie(db, nil, pool)
	trie.SetValue(nil, []byte("NilNilNilNilNil"))
	trie.SetValue([]byte("IamG"), []byte("ki10$%^%&@#!@#"))
	trie.SetValue([]byte("IamGood"), []byte("a1230xm90zm19ma"))
	trie.SetValue([]byte("tesab"), []byte("value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555"))

	trie.SetValue([]byte("tesab"), []byte("value.555val"))
	trie.SetValue([]byte("tesa"), []byte("vale....asdfasdfasdfvalue.555val"))
	trie.SetValue([]byte("tesa"), []byte("vale....asdfasdfasdfvalue.555val"))
	trie.SetValue([]byte("tes"), []byte("asdfvale....asdfasdfasdfvalue.555val"))
	trie.SetValue([]byte("tesabcd"), []byte("asdfvale....asdfasdfasdfvalue.555val"))
	trie.SetValue([]byte("t"), []byte("asdfvale....asdfasdfasdfvalue.555valasd"))
	trie.SetValue([]byte("te"), []byte("AVDED09%^$%@#@#"))

	iterator := trie.NewIterator([]byte("t"))
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}

		fmt.Printf("%s: %s\n", key, value)
	}
	fmt.Println()

	iterator1 := trie.NewIterator([]byte("te"))
	for {
		key, value, ok := iterator1.Next()
		if !ok {
			break
		}

		fmt.Printf("%s: %s\n", key, value)
	}
	fmt.Println()

	iterator2 := trie.NewIterator([]byte("I"))
	for {
		key, value, ok := iterator2.Next()
		if !ok {
			break
		}

		fmt.Printf("%s: %s\n", key, value)
	}
	fmt.Println()

	iterator3 := trie.NewIterator(nil)
	for {
		key, value, ok := iterator3.Next()
		if !ok {
			break
		}

		fmt.Printf("%s: %s\n", key, value)
	}
	fmt.Println()
}
