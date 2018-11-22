package trie_gc_unittest

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/trie_gc"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"testing"
)

func mergeHashSet(setList ...map[types.Hash]struct{}) map[types.Hash]struct{} {
	merged := make(map[types.Hash]struct{})
	for _, hashSet := range setList {
		for hash := range hashSet {
			merged[hash] = struct{}{}
		}
	}
	return merged
}

func filterHashSet(setA map[types.Hash]struct{}, setB map[types.Hash]struct{}) map[types.Hash]struct{} {
	newHashSet := make(map[types.Hash]struct{})
	for hash := range setA {
		if _, ok := setB[hash]; !ok {
			newHashSet[hash] = struct{}{}
		}
	}
	return newHashSet
}

func trieToNodeHashSet(t *trie.Trie) map[types.Hash]struct{} {
	hasSet := make(map[types.Hash]struct{})
	nodeList := t.NodeList()
	for _, node := range nodeList {
		hash := node.Hash()
		hasSet[*hash] = struct{}{}
	}
	return hasSet
}

func mark(chainInstance chain.Chain, marker *trie_gc.Marker, fromHeight uint64, toHeight uint64) uint64 {
	latestBlock := chainInstance.GetLatestSnapshotBlock()

	targetHeight := latestBlock.Height
	if targetHeight > toHeight {
		targetHeight = toHeight
	}
	marker.SetMarkedHeight(fromHeight - 1)
	marker.Mark(targetHeight, nil)

	return targetHeight
}

func getHashSet(chainInstance chain.Chain, snapshotBlocks []*ledger.SnapshotBlock) (map[types.Hash]struct{}, error) {
	hashSet := make(map[types.Hash]struct{})
	for _, snapshotBlock := range snapshotBlocks {
		stateTrie := trie.NewTrie(chainInstance.ChainDb().Db(), &snapshotBlock.StateHash, nil)
		if stateTrie == nil {
			return nil, errors.New("stateTrie is nil")
		}
		hashSet = mergeHashSet(hashSet, trieToNodeHashSet(stateTrie))

		iter := stateTrie.NewIterator(nil)
		for {
			_, value, ok := iter.Next()
			if !ok {
				break
			}

			if len(value) == 32 {
				accountHash, _ := types.BytesToHash(value)
				accountStateTrie := trie.NewTrie(chainInstance.ChainDb().Db(), &accountHash, nil)

				hashSet = mergeHashSet(hashSet, trieToNodeHashSet(accountStateTrie))

			}
		}
	}
	return hashSet, nil
}

func getDbTrieNodeNumber(chainInstance chain.Chain) uint64 {
	key, _ := database.EncodeKey(database.DBKP_TRIE_NODE)
	iter := chainInstance.ChainDb().Db().NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	count := uint64(0)
	for iter.Next() {
		count++
	}
	return count
}

func getDbHashSet(marker *trie_gc.Marker) map[types.Hash]struct{} {
	dbHashSet := make(map[types.Hash]struct{})
	dbKey, _ := database.EncodeKey(trie_gc.DBKP_MARKED_HASHLIST)

	iter := marker.Db().NewIterator(util.BytesPrefix(dbKey), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		hash, _ := types.BytesToHash(key[1:])
		dbHashSet[hash] = struct{}{}
	}
	return dbHashSet
}

func checkHashSet(t *testing.T, hashSet1 map[types.Hash]struct{}, hashSet2 map[types.Hash]struct{}) {
	hashSetLen := len(hashSet1)
	dbHashSetLen := len(hashSet2)

	if hashSetLen != dbHashSetLen {
		t.Fatal(fmt.Sprintf("hashSet length is wrong, len(hashSet) is %d, len(dbHashSet) id %d", hashSetLen, dbHashSetLen))
	}

	for hash := range hashSet1 {
		if _, ok := hashSet2[hash]; !ok {
			t.Fatal("hashSet is wrong")
		}
	}

}

func Test_marker_mark(t *testing.T) {
	dirName := "testdata"
	chainInstance := newChainInstance(dirName, false)

	marker, err := newMarkerInstance(chainInstance, dirName, true)
	if err != nil {
		t.Fatal(err)
	}

	const (
		MARK_TO_HEIGHT   = 1000
		MARK_FROM_HEIGHT = 1
	)

	targetHeight := mark(chainInstance, marker, MARK_FROM_HEIGHT, MARK_TO_HEIGHT) - 1
	count := targetHeight - MARK_FROM_HEIGHT + 1

	snapshotBlocks, err := chainInstance.GetSnapshotBlocksByHeight(targetHeight, count, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// get hash set
	hashSet, err := getHashSet(chainInstance, snapshotBlocks)
	if err != nil {
		t.Fatal(err)
	}

	// get db hash set
	dbHashSet := getDbHashSet(marker)

	markedHeight := marker.MarkedHeight()
	if markedHeight != uint64(count) {
		t.Fatal("markedHeight is wrong")
	}
	checkHashSet(t, hashSet, dbHashSet)

	fmt.Printf("hash set length is %d\n", len(hashSet))
}

func Test_marker_filter_mark(t *testing.T) {
	dirName := "testdata"
	chainInstance := newChainInstance(dirName, false)

	marker, err := newMarkerInstance(chainInstance, dirName, true)
	if err != nil {
		t.Fatal(err)
	}

	const (
		MARK_TO_HEIGHT   = 1000
		MARK_FROM_HEIGHT = 1
	)

	targetHeight := mark(chainInstance, marker, MARK_FROM_HEIGHT, MARK_TO_HEIGHT)

	// get db hash set
	dbHashSet := getDbHashSet(marker)

	snapshotBlock, err := chainInstance.GetSnapshotBlockByHeight(targetHeight)
	if err != nil {
		t.Fatal(err)
	}

	hashSet, err := getHashSet(chainInstance, []*ledger.SnapshotBlock{snapshotBlock})
	if err != nil {
		t.Fatal(err)
	}

	newHashSet := filterHashSet(dbHashSet, hashSet)

	marker.FilterMarked()
	newDbHashSet := getDbHashSet(marker)

	checkHashSet(t, newHashSet, newDbHashSet)

	fmt.Printf("Origin hash set is %d, new hash set is %d\n", len(dbHashSet), len(newDbHashSet))
}

func Test_marker_clean(t *testing.T) {
	dirName := "testdata"
	chainInstance := newChainInstance(dirName, false)

	marker, err := newMarkerInstance(chainInstance, dirName, true)
	if err != nil {
		t.Fatal(err)
	}

	const (
		MARK_TO_HEIGHT   = 100 * 10000
		MARK_FROM_HEIGHT = 1
	)
	mark(chainInstance, marker, MARK_FROM_HEIGHT, MARK_TO_HEIGHT)
	marker.FilterMarked()

	dbHashSet := getDbHashSet(marker)
	trieNodeCount := getDbTrieNodeNumber(chainInstance)

	marker.Clean(nil)
	newDbHashSet := getDbHashSet(marker)
	if len(newDbHashSet) > 0 {
		t.Fatal("clean failed")
	}

	newTrieNodeCount := getDbTrieNodeNumber(chainInstance)
	if trieNodeCount-newTrieNodeCount != uint64(len(dbHashSet)) {
		t.Fatal(fmt.Sprintf("clean error, newTrieNodeCount is %d, trieNodeCount is %d, delete count is %d, len(dbHashSet) is %d", newTrieNodeCount, trieNodeCount, trieNodeCount-newTrieNodeCount, len(dbHashSet)))
	}
}
