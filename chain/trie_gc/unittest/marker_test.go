package trie_gc_unittest

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain_db/access"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
	"testing"
)

func mergeHashSet(setList ...map[types.Hash]struct{}) map[types.Hash]struct{} {
	newHashSet := make(map[types.Hash]struct{})
	for _, hashSet := range setList {
		for hash := range hashSet {
			newHashSet[hash] = struct{}{}
		}
	}

	return newHashSet
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

func getNodeHashSet(chainInstance chain.Chain, isSnapshotBlock bool, stateHash types.Hash, hashSetCache map[types.Hash]struct{}, refHashSet map[types.Hash]struct{}, nodePool *trie.TrieNodePool) {

	if _, ok := hashSetCache[stateHash]; ok {
		return
	}

	stateTrie := trie.NewTrie(chainInstance.ChainDb().Db(), &stateHash, nodePool)
	if stateTrie == nil {
		return
	}

	inHashSet := func(node *trie.TrieNode) bool {
		if _, ok := hashSetCache[*node.Hash()]; ok {
			return false
		}
		return true
	}
	ni := stateTrie.NewNodeIterator()
	for ni.Next(inHashSet) {
		node := ni.Node()

		nodeHash := node.Hash()
		hashSetCache[*nodeHash] = struct{}{}
		if node.NodeType() == trie.TRIE_HASH_NODE {
			refHash, _ := types.BytesToHash(node.Value())
			refHashSet[refHash] = struct{}{}
		} else if node.NodeType() == trie.TRIE_VALUE_NODE {
			if isSnapshotBlock {
				value := node.Value()
				if len(value) == types.HashSize {
					accountStateHash, _ := types.BytesToHash(value)
					getNodeHashSet(chainInstance, false, accountStateHash, hashSetCache, refHashSet, nodePool)
				}
			}
		}

	}
}

func checkHashSet(t *testing.T, hashSet1 map[types.Hash]struct{}, hashSet2 map[types.Hash]struct{}) {
	hashSetLen := len(hashSet1)
	hashSetLen2 := len(hashSet2)

	if hashSetLen != hashSetLen2 {
		t.Fatal(fmt.Sprintf("hashSet length is wrong, len(hashSet1) is %d, len(hashSet2) id %d", hashSetLen, hashSetLen2))
	}

	for hash := range hashSet1 {
		if _, ok := hashSet2[hash]; !ok {
			t.Fatal("hashSet is wrong")
		}
	}
}

func mockMark(chainInstance chain.Chain) (map[types.Hash]struct{}, map[types.Hash]struct{}, error) {
	markedHashSet := make(map[types.Hash]struct{})
	refHashSet := make(map[types.Hash]struct{})
	boundarySnapshotHeight := chainInstance.GetLatestSnapshotBlock().Height - (86400 + types.AccountLimitSnapshotHeight)

	beginEventId := uint64(1)
	lastBeId, err := chainInstance.GetLatestBlockEventId()
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("lastBeId is %d\n", lastBeId)

	pool := trie.NewCustomTrieNodePool(50*10000, 25*10000)

LOOP:
	for i := lastBeId; i >= beginEventId; i-- {
		if i%10000 == 0 {
			fmt.Printf("current event id is %d, markedHashSet length is %d \n", i, len(markedHashSet))
		}
		eventType, hashList, err := chainInstance.GetEvent(i)
		if err != nil {
			return nil, nil, err
		}
		switch eventType {
		case access.AddAccountBlocksEvent:
			for _, hash := range hashList {
				block, err := chainInstance.GetAccountBlockByHash(&hash)
				if err != nil {
					return nil, nil, err
				}
				if block == nil {
					continue
				}
				getNodeHashSet(chainInstance, false, block.StateHash, markedHashSet, refHashSet, pool)

			}
		case access.AddSnapshotBlocksEvent:
			for _, hash := range hashList {
				block, err := chainInstance.GetSnapshotBlockByHash(&hash)
				if err != nil {
					return nil, nil, err
				}
				if block == nil {
					continue
				}

				getNodeHashSet(chainInstance, true, block.StateHash, markedHashSet, refHashSet, pool)

				if block.Height < boundarySnapshotHeight {
					break LOOP
				}
			}
		}

	}

	return markedHashSet, refHashSet, nil
}

func loadAllHashSet(chainInstance chain.Chain) map[types.Hash]struct{} {
	allHashSet := make(map[types.Hash]struct{})
	key, _ := database.EncodeKey(database.DBKP_TRIE_NODE)
	iter := chainInstance.ChainDb().Db().NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	for iter.Next() {
		key, _ := types.BytesToHash(iter.Key()[1:])
		allHashSet[key] = struct{}{}
	}
	return allHashSet
}

func loadAllRefHashSet(chainInstance chain.Chain) map[types.Hash]struct{} {
	allHashSet := make(map[types.Hash]struct{})
	key, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE)
	iter := chainInstance.ChainDb().Db().NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	for iter.Next() {
		key, _ := types.BytesToHash(iter.Key()[1:])
		allHashSet[key] = struct{}{}
	}
	return allHashSet
}

func Test_marker_mark_clean(t *testing.T) {
	dirName := "testdata"
	chainInstance := newChainInstance(dirName, false)
	marker := newMarkerInstance(chainInstance)

	fmt.Printf("loadAllHashSet...\n")
	allHashSet := loadAllHashSet(chainInstance)
	fmt.Printf("allHashSet length is %d...\n", len(allHashSet))

	fmt.Printf("loadAllRefHashSet...\n")
	allRefHashSet := loadAllRefHashSet(chainInstance)
	fmt.Printf("allRefHashSet length is %d...\n", len(allRefHashSet))

	fmt.Printf("mockMark...\n")
	markedHashSetByMock, refHashSetByMock, err := mockMark(chainInstance)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("marker.MarkAndClean...\n")
	err2 := marker.MarkAndClean(nil)
	if err2 != nil {
		t.Fatal(err2)
	}

	fmt.Printf("loadNewAllHashSet...\n")
	newAllHashSet := loadAllHashSet(chainInstance)

	fmt.Printf("loadNewAllRefHashSet...\n")
	newAllRefHashSet := loadAllRefHashSet(chainInstance)

	fmt.Printf("checkHashSet...\n")
	checkHashSet(t, newAllHashSet, markedHashSetByMock)

	fmt.Printf("checkRefHashSet...\n")
	checkHashSet(t, newAllRefHashSet, refHashSetByMock)

	cleanHashSet := filterHashSet(allHashSet, newAllHashSet)
	fmt.Printf("Clean %d trie nodes\n", len(cleanHashSet))

	cleanRefHashSet := filterHashSet(allRefHashSet, newAllRefHashSet)
	fmt.Printf("Clean %d ref nodes\n", len(cleanRefHashSet))
}
