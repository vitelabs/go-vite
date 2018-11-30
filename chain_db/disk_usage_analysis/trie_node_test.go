package disk_usage_analysis

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/trie"
	"testing"
)

func Test_trieNode(t *testing.T) {
	chainDb := newChainDb("testdata")
	iter := chainDb.Db().NewIterator(util.BytesPrefix([]byte{database.DBKP_TRIE_NODE}), nil)
	defer iter.Release()

	for iter.Next() {
		node := &trie.TrieNode{}
		length := len(iter.Value())
		if length > 1024 {
			node.DbDeserialize(iter.Value())
			fmt.Printf("byte length: %d, children count: %d\n", length, len(node.Children()))
		}
	}
}

func Test_currentTrie(t *testing.T) {
	chainDb := newChainDb("testdata")
	iter := chainDb.Db().NewIterator(util.BytesPrefix([]byte{database.DBKP_TRIE_NODE}), nil)
	defer iter.Release()

	latestBlock, _ := chainDb.Sc.GetLatestBlock()
	tr := trie.NewTrie(chainDb.Db(), &latestBlock.StateHash, nil)
	iterator := tr.NewIterator(nil)
	var result [20]map[byte]uint64
	var diskUsage int
	count := 0
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		diskUsage += len(key)
		diskUsage += len(value)

		count++
		for index, aByte := range key {
			if result[index] == nil {
				result[index] = make(map[byte]uint64)
			}
			result[index][aByte]++
		}
	}
	fmt.Printf("totalKey: %d\n", count)
	fmt.Printf("disk usage: %d Byte\n", diskUsage)

	for index, aMap := range result {
		fmt.Printf("%d: %d\n", index, len(aMap))
	}

}
