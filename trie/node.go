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

func (trieNode *TrieNode) Copy() *TrieNode {
	return &TrieNode{
		hash:     trieNode.hash,
		nodeType: trieNode.nodeType,
		children: trieNode.children,
		key:      trieNode.key,
		value:    trieNode.value,
	}
}

func (trieNode *TrieNode) Hash() *types.Hash {
	return trieNode.hash
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
