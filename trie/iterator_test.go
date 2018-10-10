package trie

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewIterator(t *testing.T) {
	db, _ := database.NewLevelDb(filepath.Join(common.GoViteTestDataDir(), "trie"))
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

type Cmap struct {
	m2 map[int]int
}

func TestConcurrence(t *testing.T) {
	m1 := map[int][]byte{
		3: {5},
		4: {6},
	}
	cm1 := Cmap{
		m2: map[int]int{
			3: 5,
			4: 6,
		},
	}

	for z := 0; z < 5; z++ {
		var sw sync.WaitGroup
		for i := 0; i < 10; i++ {
			sw.Add(1)
			go func() {
				defer sw.Done()

				for j := 0; j < 1000000; j++ {
					fmt.Sprint(m1[3])
					fmt.Sprint(m1[4])
				}
			}()
		}
		sw.Wait()
	}

	for z := 0; z < 5; z++ {
		var sw sync.WaitGroup
		for i := 0; i < 10; i++ {
			sw.Add(1)
			go func() {
				defer sw.Done()

				for j := 0; j < 1000000; j++ {
					cm2 := cm1
					fmt.Sprint(cm2.m2[3])
				}
			}()
		}
		sw.Wait()
	}

}
