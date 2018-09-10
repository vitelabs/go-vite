package trie

import "github.com/vitelabs/go-vite/common/types"

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

func (trieNode *TrieNode) Copy() *TrieNode {
	return &TrieNode{
		hash:     trieNode.hash,
		nodeType: trieNode.nodeType,
		children: trieNode.children,
		key:      trieNode.key,
		value:    trieNode.value,
	}
}

func (trieNode *TrieNode) ComputeHash() *types.Hash {
	return nil
}

func (trieNode *TrieNode) SetHash(hash *types.Hash) {
	trieNode.hash = hash
}

func (trieNode *TrieNode) Hash() *types.Hash {
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
