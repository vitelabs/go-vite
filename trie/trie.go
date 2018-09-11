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

	RootHash *types.Hash
	Root     *TrieNode

	unSavedRefValueMap map[types.Hash][]byte
}

func NewTrie(db *leveldb.DB, rootHash *types.Hash, pool *TrieNodePool) (*Trie, error) {
	trie := &Trie{
		db:       db,
		RootHash: rootHash,
		log:      log15.New("module", "trie"),

		unSavedRefValueMap: make(map[types.Hash][]byte),
	}

	trie.loadFromDb()
	return trie, nil
}

func (trie *Trie) getNodeFromDb(key *types.Hash) *TrieNode {
	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, key.Bytes())
	value, err := trie.db.Get(dbKey, nil)
	if err != nil {
		trie.log.Error("Query trie node failed from the database, error is "+err.Error(), "method", "getNodeFromDb")
		return nil
	}
	trieNode := &TrieNode{}
	dsErr := trieNode.DbDeserialize(value)
	if dsErr != nil {
		trie.log.Error("Deserialize trie node  failed, error is "+err.Error(), "method", "getNodeFromDb")
		return nil
	}

	return trieNode
}

func (trie *Trie) saveNodeInDb(batch *leveldb.Batch, node *TrieNode) error {
	if node.Hash() == nil {
		return errors.New("saveNodeInDb() failed, because node.Hash() is nil.")
	}
	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_NODE, node.Hash())
	cachedNode := trie.getNode(node.Hash())
	if cachedNode != nil {
		return nil
	}

	data, err := node.DbSerialize()

	if err != nil {
		return errors.New("DbSerialize trie node failed, error is " + err.Error())
	}

	batch.Put(dbKey, data)
	return nil
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

	dbKey, _ := database.EncodeKey(database.DBKP_TRIE_REF_VALUE, key)
	return trie.db.Get(dbKey, nil)
}

func (trie *Trie) getNode(key *types.Hash) *TrieNode {
	node := trie.cachePool.Get(key)
	if node != nil {
		return node
	}

	node = trie.getNodeFromDb(key)
	if node != nil {
		trie.cachePool.Set(key, node)
	}
	return node
}

func (trie *Trie) loadFromDb() {
	if trie.RootHash == nil {
		return
	}

	trie.Root = trie.traverseLoad(trie.RootHash)
}

func (trie *Trie) traverseLoad(hash *types.Hash) *TrieNode {
	node := trie.getNode(hash)
	if node == nil {
		return nil
	}

	switch node.NodeType() {
	case TRIE_FULL_NODE:
		for key, child := range node.children {
			node.children[key] = trie.traverseLoad(child.Hash())
		}
	case TRIE_SHORT_NODE:
		node.child = trie.traverseLoad(node.child.Hash())
	}
	return node
}

func (trie *Trie) computeHash() {

}

func (trie *Trie) Copy() *Trie {
	return &Trie{
		db:        trie.db,
		cachePool: trie.cachePool,
		log:       trie.log,

		RootHash: trie.RootHash,
		Root:     trie.Root.Copy(),
	}
}

func (trie *Trie) Save() error {
	batch := new(leveldb.Batch)
	err := trie.traverseSave(batch, trie.Root)
	if err != nil {
		return err
	}

	trie.saveRefValueMap(batch)

	writeErr := trie.db.Write(batch, nil)
	if writeErr != nil {
		return writeErr
	}

	trie.unSavedRefValueMap = make(map[types.Hash][]byte)
	return nil

}

func (trie *Trie) traverseSave(batch *leveldb.Batch, node *TrieNode) error {
	if node == nil {
		return nil
	}

	err := trie.saveNodeInDb(batch, node)
	if err != nil {
		return err
	}

	switch node.NodeType() {
	case TRIE_FULL_NODE:
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
		newNode := node.Copy()
		newNode.SetHash(nil)

		if len(key) > 0 {
			firstChar := key[0]
			newNode.children[firstChar] = trie.setValue(newNode.children[firstChar], key[1:], leafNode)
		} else {
			newNode.children[byte(0)] = leafNode
		}
		return node
	case TRIE_SHORT_NODE:

		var keyChar byte
		var restKey []byte
		var index = 0

		for ; index < len(key); index++ {
			char := key[index]
			if index >= len(node.key) ||
				node.key[index] != char {
				keyChar = char
				restKey = key[index+1:]
				break
			}

		}

		var fullNode *TrieNode

		if index >= len(node.key) {
			if len(key) == index &&
				node.child.NodeType() == TRIE_VALUE_NODE ||
				node.child.NodeType() == TRIE_HASH_NODE {

				// Hash is not correct, so clear
				node.SetHash(nil)
				node.SetChild(leafNode)
				return node
			} else if node.child.NodeType() == TRIE_FULL_NODE {
				fullNode = node.child.Copy()
				// Hash is not correct, so clear
				fullNode.SetHash(nil)
			}
		}

		if fullNode == nil {
			fullNode = NewFullNode(nil)
			var nodeChar byte
			var nodeRestKey []byte
			if index < len(node.key) {
				nodeChar = node.key[index]
				nodeRestKey = node.key[index+1:]
			}

			fullNode.children[nodeChar] = trie.setValue(fullNode.children[nodeChar], nodeRestKey, node.child)
		}

		fullNode.children[keyChar] = trie.setValue(fullNode.children[keyChar], restKey, leafNode)
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
			fullNode.children[byte(0)] = node
			fullNode.children[key[0]] = trie.setValue(nil, key[1:], leafNode)
			return fullNode
		} else {
			if node.NodeType() == TRIE_HASH_NODE {
				valueHash, err := types.BytesToHash(node.value)
				if err == nil {
					if _, ok := trie.unSavedRefValueMap[valueHash]; ok {
						delete(trie.unSavedRefValueMap, valueHash)
					}
				}
			}
			return leafNode
		}
	}

	return nil
}

func (trie *Trie) GetValue(key []byte) []byte {

	leafNode := trie.getLeafNode(trie.Root, key)
	if leafNode == nil {
		return nil
	}

	switch leafNode.NodeType() {
	case TRIE_VALUE_NODE:
		return leafNode.value

	case TRIE_HASH_NODE:
		value, _ := trie.getRefValue(leafNode.value)
		return value
	}

	return nil
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
			return node.children[byte(0)]
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
