package trie

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
)

type Trie struct {
	db        *leveldb.DB
	cachePool *TrieNodePool
	log       log15.Logger

	Root *TrieNode

	unSavedRefValueMap map[types.Hash][]byte
}

func DeleteNodes(db *leveldb.DB, hashList []types.Hash) error {
	batch := new(leveldb.Batch)
	for _, hash := range hashList {
		dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, hash.Bytes())
		batch.Delete(dbKey)
	}
	return db.Write(batch, nil)
}

func NewTrie(db *leveldb.DB, rootHash *types.Hash, pool *TrieNodePool) *Trie {
	trie := &Trie{
		db:        db,
		cachePool: pool,
		log:       log15.New("module", "trie"),

		unSavedRefValueMap: make(map[types.Hash][]byte),
	}

	trie.loadFromDb(rootHash)
	return trie
}

func (trie *Trie) getNodeFromDb(key *types.Hash) *TrieNode {
	if trie.db == nil {
		return nil
	}
	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, key.Bytes())
	value, err := trie.db.Get(dbKey, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			trie.log.Error("Query trie node failed from the database, error is "+err.Error(), "method", "getNodeFromDb")
		}

		return nil
	}
	trieNode := &TrieNode{}
	dsErr := trieNode.DbDeserialize(value)
	if dsErr != nil {
		trie.log.Error("Deserialize trie node  failed, error is "+dsErr.Error(), "method", "getNodeFromDb")
		return nil
	}

	return trieNode
}

func (trie *Trie) saveNodeInDb(batch *leveldb.Batch, node *TrieNode) error {
	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, node.Hash().Bytes())
	data, err := node.DbSerialize()

	if err != nil {
		return errors.New("DbSerialize trie node failed, error is " + err.Error())
	}

	batch.Put(dbKey, data)
	return nil
}

func (trie *Trie) deleteUnSavedRefValueMap(node *TrieNode) {
	if node == nil ||
		node.NodeType() != TRIE_HASH_NODE {
		return
	}

	valueHash, err := types.BytesToHash(node.value)
	if err != nil {
		return
	}
	if _, ok := trie.unSavedRefValueMap[valueHash]; ok {
		delete(trie.unSavedRefValueMap, valueHash)
	}

}

func (trie *Trie) saveRefValueMap(batch *leveldb.Batch) {
	for key, value := range trie.unSavedRefValueMap {
		dbKey, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE, key.Bytes())
		batch.Put(dbKey, value)
	}
}

func (trie *Trie) getRefValue(key []byte) ([]byte, error) {
	hashKey, err := types.BytesToHash(key)
	if err != nil {
		return nil, err
	}

	if value, ok := trie.unSavedRefValueMap[hashKey]; ok {
		return value, nil
	}

	if trie.db == nil {
		return nil, nil
	}

	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE, key)
	return trie.db.Get(dbKey, nil)
}

func (trie *Trie) getNode(key *types.Hash) *TrieNode {

	if trie.cachePool != nil {
		node := trie.cachePool.Get(key)
		if node != nil {
			return node
		}
	}

	node := trie.getNodeFromDb(key)
	if node != nil {
		if trie.cachePool != nil {
			trie.cachePool.Set(key, node)
		}
	}
	return node
}

func (trie *Trie) loadFromDb(rootHash *types.Hash) {
	if rootHash == nil {
		return
	}

	trie.Root = trie.traverseLoad(rootHash)
}

func (trie *Trie) traverseLoad(hash *types.Hash) *TrieNode {
	node := trie.getNode(hash)
	if node == nil {
		return nil
	}

	switch node.NodeType() {
	case TRIE_FULL_NODE:
		node.AtomicComplete(func() {
			for key, child := range node.children {
				node.children[key] = trie.traverseLoad(child.Hash())
			}
			if node.child != nil {
				node.child = trie.traverseLoad(node.child.Hash())
			}
		})
	case TRIE_SHORT_NODE:
		node.child = trie.traverseLoad(node.child.Hash())
	}
	return node
}

func (trie *Trie) Hash() *types.Hash {
	if trie.Root == nil {
		return nil
	}

	return trie.Root.Hash()
}

func (trie *Trie) Copy() *Trie {
	newTrie := &Trie{
		db:        trie.db,
		cachePool: trie.cachePool,
		log:       trie.log,

		unSavedRefValueMap: make(map[types.Hash][]byte),
	}
	if trie.Root != nil {
		newTrie.Root = trie.Root.Copy(true)
	}
	return newTrie
}

func (trie *Trie) Save(batch *leveldb.Batch) (successCallback func(), returnErr error) {
	err := trie.traverseSave(batch, trie.Root)
	if err != nil {
		return nil, err
	}

	trie.saveRefValueMap(batch)

	return func() {
		trie.unSavedRefValueMap = make(map[types.Hash][]byte)
	}, nil
}

func (trie *Trie) traverseSave(batch *leveldb.Batch, node *TrieNode) error {
	if node == nil {
		return nil
	}

	// Cached, no save
	if trie.getNode(node.Hash()) != nil {
		return nil
	}

	err := trie.saveNodeInDb(batch, node)
	if err != nil {
		return err
	}

	switch node.NodeType() {
	case TRIE_FULL_NODE:
		if node.child != nil {
			trie.traverseSave(batch, node.child)
		}

		for _, child := range node.children {
			trie.traverseSave(batch, child)
		}

	case TRIE_SHORT_NODE:
		trie.traverseSave(batch, node.child)
	}
	return nil
}

func (trie *Trie) SetValue(key []byte, value []byte) {
	var leafNode *TrieNode
	if len(value) > 32 {
		valueHash, _ := types.BytesToHash(crypto.Hash256(value))
		leafNode = NewHashNode(&valueHash)
		defer func() {
			trie.unSavedRefValueMap[valueHash] = value
		}()
	} else {
		leafNode = NewValueNode(value)

	}

	trie.Root = trie.setValue(trie.Root, key, leafNode)
}

func (trie *Trie) setValue(node *TrieNode, key []byte, leafNode *TrieNode) *TrieNode {
	// Create short_node when node is nil
	if node == nil {
		if len(key) != 0 {
			shortNode := NewShortNode(key, nil)
			shortNode.SetChild(leafNode)
			return shortNode
		} else {
			// Final node
			return leafNode
		}

	}

	// Normal node
	switch node.NodeType() {
	case TRIE_FULL_NODE:
		// Hash is not correct, so clear
		newNode := node.Copy(false)

		if len(key) > 0 {
			firstChar := key[0]
			newNode.children[firstChar] = trie.setValue(newNode.children[firstChar], key[1:], leafNode)
		} else {
			trie.deleteUnSavedRefValueMap(newNode.child)
			newNode.child = leafNode
		}
		return newNode
	case TRIE_SHORT_NODE:
		// sometimes is nil
		var keyChar *byte
		var restKey []byte

		var index = 0
		for ; index < len(key); index++ {
			char := key[index]
			if index >= len(node.key) || node.key[index] != char {
				keyChar = &char
				restKey = key[index+1:]
				break
			}

		}

		var fullNode *TrieNode

		if index >= len(node.key) {
			if len(key) == index && (node.child.NodeType() == TRIE_VALUE_NODE || node.child.NodeType() == TRIE_HASH_NODE) {
				trie.deleteUnSavedRefValueMap(node.child)

				newNode := node.Copy(false)
				newNode.SetChild(leafNode)

				return newNode
			} else if node.child.NodeType() == TRIE_FULL_NODE {
				fullNode = node.child.Copy(false)
			}
		}

		if fullNode == nil {
			fullNode = NewFullNode(nil)
			// sometimes is nil
			var nodeChar *byte
			var nodeRestKey []byte
			if index < len(node.key) {
				nodeChar = &node.key[index]
				nodeRestKey = node.key[index+1:]
			}

			if nodeChar != nil {
				fullNode.children[*nodeChar] = trie.setValue(fullNode.children[*nodeChar], nodeRestKey, node.child)
			} else {
				fullNode.child = node.child
			}
		}

		if keyChar != nil {
			fullNode.children[*keyChar] = trie.setValue(fullNode.children[*keyChar], restKey, leafNode)
		} else {
			fullNode.child = leafNode
		}
		if index > 0 {
			shortNode := NewShortNode(key[0:index], nil)
			shortNode.SetChild(fullNode)
			return shortNode
		} else {
			return fullNode
		}
	default:
		if len(key) > 0 {
			fullNode := NewFullNode(nil)
			fullNode.child = node
			fullNode.children[key[0]] = trie.setValue(nil, key[1:], leafNode)
			return fullNode
		} else {
			trie.deleteUnSavedRefValueMap(node)
			return leafNode
		}
	}

	return nil
}

func (trie *Trie) LeafNodeValue(leafNode *TrieNode) []byte {
	if leafNode == nil {
		return nil
	}

	switch leafNode.NodeType() {
	case TRIE_VALUE_NODE:
		return leafNode.value

	case TRIE_HASH_NODE:
		value, _ := trie.getRefValue(leafNode.value)
		return value
	default:
		return nil
	}
}

func (trie *Trie) GetValue(key []byte) []byte {

	leafNode := trie.getLeafNode(trie.Root, key)

	return trie.LeafNodeValue(leafNode)
}

func (trie *Trie) NewNodeIterator() *NodeIterator {
	return NewNodeIterator(trie)
}

func (trie *Trie) NewIterator(prefix []byte) *Iterator {
	return NewIterator(trie, prefix)
}

func (trie *Trie) getLeafNode(node *TrieNode, key []byte) *TrieNode {
	if node == nil {
		return nil
	}

	if len(key) == 0 {
		switch node.NodeType() {
		case TRIE_HASH_NODE:
			fallthrough
		case TRIE_VALUE_NODE:
			return node
		case TRIE_FULL_NODE:
			return node.child
		default:
			return nil
		}
	}

	switch node.NodeType() {
	case TRIE_FULL_NODE:
		return trie.getLeafNode(node.children[key[0]], key[1:])
	case TRIE_SHORT_NODE:
		if !bytes.HasPrefix(key, node.key) {
			return nil
		}
		return trie.getLeafNode(node.child, key[len(node.key):])
	default:
		return nil
	}
}
