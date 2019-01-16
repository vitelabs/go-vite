package trie_gc_unittest

import (
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/trie_gc"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm"
	"log"
	"net/http"
	_ "net/http/pprof"
	"testing"
)

func deleteAllTrie(chainInstance chain.Chain) error {
	batch := new(leveldb.Batch)

	// clear trie node
	dbkey, _ := database.EncodeKey(database.DBKP_TRIE_NODE)
	iter := chainInstance.TrieDb().NewIterator(util.BytesPrefix(dbkey), nil)
	defer iter.Release()

	for iter.Next() {
		batch.Delete(iter.Key())
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}

	// clear ref value
	refDbKey, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE)
	refIter := chainInstance.TrieDb().NewIterator(util.BytesPrefix(refDbKey), nil)
	defer refIter.Release()
	for refIter.Next() {
		batch.Delete(refIter.Key())

	}
	if err := refIter.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}

	if err := chainInstance.ChainDb().Commit(batch); err != nil {
		return err
	}
	return nil
}

func recoverTrie(chainInstance chain.Chain) error {
	collector := trie_gc.NewCollector(chainInstance, 0)
	return collector.Recover()
}

func isNodeExist(chainInstance chain.Chain, node *trie.TrieNode) (bool, error) {
	db := chainInstance.ChainDb().Db()

	nodeHash := node.Hash()
	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, nodeHash.Bytes())
	ok, err := db.Has(dbKey, nil)
	if !ok || err != nil {
		return ok, err
	}

	if node.NodeType() == trie.TRIE_HASH_NODE {
		dbKey2, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE, node.Value())
		ok2, err := db.Has(dbKey2, nil)
		if !ok || err != nil {
			return ok2, err
		}
	}
	return true, nil

}

func checkTrie(chainInstance chain.Chain) error {
	const (
		checkSnapshotBlockNum = uint64(86400 * 2)
		numPerCheck           = 100
	)

	latestSnapshotBlock := chainInstance.GetLatestSnapshotBlock()
	startSnapshotBlockHeight := uint64(1)

	if latestSnapshotBlock.Height > checkSnapshotBlockNum {
		startSnapshotBlockHeight = latestSnapshotBlock.Height - checkSnapshotBlockNum + 1
	}

	fmt.Printf("check from %d to %d\n", startSnapshotBlockHeight, latestSnapshotBlock.Height)

	current := startSnapshotBlockHeight

	checkedHashSet := make(map[types.Hash]struct{})
	isDeepInto := func(node *trie.TrieNode) bool {
		nodeHash := node.Hash()
		if nodeHash == nil {
			return false
		}

		if _, ok := checkedHashSet[*nodeHash]; ok {
			return false
		}
		return true
	}
	for {
		if current > latestSnapshotBlock.Height {
			break
		}
		next := current + numPerCheck

		if next > latestSnapshotBlock.Height {
			next = latestSnapshotBlock.Height
		}
		sbList, accountBlocks, err := chainInstance.GetConfirmSubLedger(current, next)
		if err != nil {
			return err
		}

		for _, sb := range sbList {
			t := chainInstance.GetStateTrie(&sb.StateHash)
			iter := t.NewNodeIterator()
			for iter.Next(isDeepInto) {
				currentNode := iter.Node()
				if ok, err := isNodeExist(chainInstance, currentNode); err != nil {
					return err
				} else if !ok {
					err := errors.New(fmt.Sprintf("node is not exist. snapshot block hash is %s, snapshot block height is %d", sb.Hash, sb.Height))
					return err
				}
				checkedHashSet[*currentNode.Hash()] = struct{}{}
			}

		}

		for _, blocks := range accountBlocks {
			for _, block := range blocks {
				t := chainInstance.GetStateTrie(&block.StateHash)
				iter := t.NewNodeIterator()
				for iter.Next(isDeepInto) {
					currentNode := iter.Node()
					if ok, err := isNodeExist(chainInstance, currentNode); err != nil {
						return err
					} else if !ok {
						err := errors.New(fmt.Sprintf("node is not exist. account block hash is %s, account block height is %d", block.Hash, block.Height))
						return err
					}
					checkedHashSet[*currentNode.Hash()] = struct{}{}
				}
			}
		}

		current = next + 1
	}
	return nil
}

func Test_recover(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:8080", nil))
	}()

	vm.InitVmConfig(false, false, false, "")

	dirName := "testdata"
	chainInstance := newChainInstance(dirName, false)

	// delete all trie
	fmt.Printf("delete trie...\n")
	if err := deleteAllTrie(chainInstance); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("complete delete trie...\n")

	// recover all trie
	fmt.Printf("recovering trie...\n")
	if err := recoverTrie(chainInstance); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("complete recover trie...\n")

	// check all trie
	fmt.Printf("checking trie...\n")
	if err := checkTrie(chainInstance); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("complete check trie...\n")
}
