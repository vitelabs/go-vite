package trie

import "github.com/vitelabs/go-vite/common/types"

type TrieNode struct {
	hash *types.Hash

	NodeType byte
	Value    []*TrieValue
}

func (trieNode *TrieNode) Copy() *TrieNode {
	return &TrieNode{
		NodeType: trieNode.NodeType,
		hash:     trieNode.hash,
		Value:    trieNode.Value[0:],
	}
}

func (*TrieNode) DbSerialize() ([]byte, error) {
	return nil, nil
}

func (*TrieNode) DbDeserialize([]byte) error {
	return nil
}

type TrieValue struct {
	Key   []byte
	Value []*TrieNode
}
