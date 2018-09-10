package trie

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common"
	"path/filepath"
	"testing"
)

func TestNewTrie(t *testing.T) {
	db := database.NewLevelDb(filepath.Join(common.GoViteTestDataDir(), "trie"))
	defer db.Close()

	pool := NewTrieNodePool()

	trie, ntErr := NewTrie(db, nil, pool)
	if ntErr != nil {
		t.Fatal(ntErr)
	}

	trie.SetValue([]byte("test"), []byte("value.hash"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.child)
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Println()

	trie.SetValue([]byte("tesa"), []byte("value.hash2"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.child.children[byte('t')])
	fmt.Printf("%+v\n", trie.Root.child.children[byte('a')])
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesa")))
	fmt.Println()

	trie.SetValue([]byte("aofjas"), []byte("value.content1"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.children['a'])
	fmt.Printf("%+v\n", trie.Root.children['t'])
	fmt.Printf("%+v\n", trie.Root.children['t'].key)
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('t')])
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('a')])
	fmt.Printf("%+v\n", trie.Root.children['a'].child)
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesa")))
	fmt.Printf("%s\n", trie.GetValue([]byte("aofjas")))
	fmt.Println()

	trie.SetValue([]byte("aofjas"), []byte("value.content2value.content2value.content2value.content2value.content2value.content2value.content2value.content2"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.children['a'])
	fmt.Printf("%+v\n", trie.Root.children['t'])
	fmt.Printf("%+v\n", trie.Root.children['t'].key)
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('t')])
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('a')])
	fmt.Printf("%+v\n", trie.Root.children['a'].child)
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesa")))
	fmt.Printf("%s\n", trie.GetValue([]byte("aofjas")))
	fmt.Println()

	trie.SetValue([]byte("tesa"), []byte("value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3value.hash3"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.children['a'])
	fmt.Printf("%+v\n", trie.Root.children['t'])
	fmt.Printf("%+v\n", trie.Root.children['t'].key)
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('t')])
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('a')])
	fmt.Printf("%+v\n", trie.Root.children['a'].child)
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesa")))
	fmt.Printf("%s\n", trie.GetValue([]byte("aofjas")))
	fmt.Println()

	trie.SetValue([]byte("tesabcd"), []byte("value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4value.hash4"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.children['a'])
	fmt.Printf("%+v\n", trie.Root.children['t'])
	fmt.Printf("%+v\n", trie.Root.children['t'].key)
	fmt.Printf("%+v\n", trie.Root.children['t'].child)
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('t')])
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('a')])
	fmt.Printf("%+v\n", trie.Root.children['t'].child.children[byte('a')].children[byte(0)])
	fmt.Printf("%+v\n", trie.Root.children['a'].child)
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesa")))
	fmt.Printf("%s\n", trie.GetValue([]byte("aofjas")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesabcd")))
	fmt.Println()

	trie.SetValue([]byte("tesab"), []byte("value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555"))
	fmt.Printf("%s\n", trie.GetValue([]byte("test")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesa")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesab")))
	fmt.Printf("%s\n", trie.GetValue([]byte("aofjas")))
	fmt.Printf("%s\n", trie.GetValue([]byte("tesabcd")))

	fmt.Println()
}

func TestNewTrie2(t *testing.T) {
	db := database.NewLevelDb(filepath.Join(common.GoViteTestDataDir(), "trie"))
	defer db.Close()

	pool := NewTrieNodePool()

	trie, ntErr := NewTrie(db, nil, pool)
	if ntErr != nil {
		t.Fatal(ntErr)
	}

	trie.SetValue([]byte("test"), []byte("value.hash"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.child)
	fmt.Println()

	trie.SetValue([]byte("testabcdef"), []byte("value.hash2"))
	fmt.Printf("%+v\n", trie.Root)
	fmt.Printf("%+v\n", trie.Root.child.children[byte(0)])
	fmt.Printf("%+v\n", trie.Root.child.children[byte('a')])
	fmt.Printf("%+v\n", trie.Root.child.children[byte('a')].child)
	fmt.Println()
}
