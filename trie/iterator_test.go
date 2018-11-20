package trie

import (
	"fmt"
	"testing"
)

func TestNewIterator(t *testing.T) {
	trie, _, close := getTrieOfNewContext()
	defer close()

	var key1 []byte
	key2 := []byte("IamG")
	key3 := []byte("IamGood")
	key4 := []byte("tesab")
	key5 := []byte("tesa")
	key6 := []byte("tes")
	key7 := []byte("tesabcd")
	key8 := []byte("t")
	key9 := []byte("te")

	value1 := []byte("NilNilNilNilNil")
	value2 := []byte("ki10$%^%&@#!@#")
	value3 := []byte("a1230xm90zm19ma")
	value4 := []byte("value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555value.555")
	value5 := []byte("value.555val")
	value6 := []byte("vale....asdfasdfasdfvalue.555val")
	value7 := []byte("asdfvale....asdfasdfasdfvalue.555val")
	value8 := []byte("asdfvale....asdfasdfasdfvalue.555valasd")
	value9 := []byte("AVDED09%^$%@#@#")

	trie.SetValue(key1, value1)
	trie.SetValue(key2, value2)
	trie.SetValue(key3, value3)
	trie.SetValue(key4, value4)

	trie.SetValue(key4, value5)
	trie.SetValue(key5, value6)
	trie.SetValue(key5, value6)
	trie.SetValue(key6, value7)
	trie.SetValue(key7, value7)
	trie.SetValue(key8, value8)
	trie.SetValue(key9, value9)

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
