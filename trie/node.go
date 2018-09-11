package trie

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
)

const (
	TRIE_FULL_NODE = byte(iota)
	TRIE_SHORT_NODE
	TRIE_VALUE_NODE
	TRIE_HASH_NODE
)

type TrieNode struct {
	hash     *types.Hash
	nodeType byte

	// fullNode
	children map[byte]*TrieNode

	// shortNode
	key   []byte
	child *TrieNode

	// hashNode and valueNode
	value []byte
}

func NewFullNode(children map[byte]*TrieNode) *TrieNode {
	if children == nil {
		children = make(map[byte]*TrieNode)
	}
	node := &TrieNode{
		children: children,
		nodeType: TRIE_FULL_NODE,
	}

	return node
}

func NewShortNode(key []byte, child *TrieNode) *TrieNode {
	node := &TrieNode{
		key:   key,
		child: child,

		nodeType: TRIE_SHORT_NODE,
	}

	return node
}

func NewHashNode(hash *types.Hash) *TrieNode {
	node := &TrieNode{
		value:    hash.Bytes(),
		nodeType: TRIE_HASH_NODE,
	}

	return node
}

func NewValueNode(value []byte) *TrieNode {
	node := &TrieNode{
		value:    value,
		nodeType: TRIE_VALUE_NODE,
	}

	return node
}

func (trieNode *TrieNode) Copy(copyHash bool) *TrieNode {
	newNode := &TrieNode{
		nodeType: trieNode.nodeType,
		children: trieNode.children,
		key:      trieNode.key,
		value:    trieNode.value,
	}
	if copyHash {
		newNode.hash = trieNode.hash
	}
	return newNode
}

func (trieNode *TrieNode) Hash() *types.Hash {
	if trieNode.hash == nil {
		var source []byte
		switch trieNode.NodeType() {
		case TRIE_FULL_NODE:
			source = []byte{TRIE_FULL_NODE}

			sc := newSortedChildren(trieNode.children)
			for _, c := range sc {
				source = append(source, c.Key)
				source = append(source, c.Value.Hash().Bytes()...)
			}
		case TRIE_SHORT_NODE:
			source = []byte{TRIE_SHORT_NODE}
			source = append(source, trieNode.key[:]...)
			source = append(source, trieNode.child.Hash().Bytes()...)
		case TRIE_HASH_NODE:
			source = []byte{TRIE_HASH_NODE}
			source = trieNode.value
		case TRIE_VALUE_NODE:
			source = []byte{TRIE_VALUE_NODE}
			source = trieNode.value
		}

		hash, _ := types.BytesToHash(crypto.Hash256(source))
		trieNode.hash = &hash
	}
	return trieNode.hash
}

func (trieNode *TrieNode) SetChild(child *TrieNode) {
	if trieNode.NodeType() == TRIE_SHORT_NODE {
		trieNode.child = child
	}
}

func (trieNode *TrieNode) NodeType() byte {
	return trieNode.nodeType
}

func (*TrieNode) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*TrieNode) DbDeserialize([]byte) error {
	return nil
}
