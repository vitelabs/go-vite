package trie

import (
	"bytes"
)

type middleKeyAndNode struct {
	key        []byte
	middleNode *TrieNode
}

type leafKeyAndNode struct {
	key      []byte
	leafNode *TrieNode
}

type Iterator struct {
	prefix []byte
	trie   *Trie

	middleNodes []*middleKeyAndNode
	leafNodes   []*leafKeyAndNode
}

func NewIterator(trie *Trie, prefix []byte) *Iterator {
	return &Iterator{
		trie:   trie,
		prefix: prefix,
		middleNodes: []*middleKeyAndNode{{
			key:        []byte{},
			middleNode: trie.Root,
		}},

		leafNodes: make([]*leafKeyAndNode, 0),
	}
}

func (iterator *Iterator) Next() (key, value []byte, ok bool) {
	for {
		if len(iterator.leafNodes) > 0 {
			node := iterator.leafNodes[0]
			iterator.leafNodes = iterator.leafNodes[1:]
			return node.key, iterator.trie.LeafNodeValue(node.leafNode), true
		}

		if len(iterator.middleNodes) <= 0 {
			return nil, nil, false
		}

		node := iterator.middleNodes[0]
		iterator.middleNodes = iterator.middleNodes[1:]

		var keys [][]byte
		var children []*TrieNode
		switch node.middleNode.NodeType() {
		case TRIE_FULL_NODE:
			for key, childNode := range node.middleNode.children {

				keys = append(keys, []byte{key})
				children = append(children, childNode)
			}
		case TRIE_SHORT_NODE:
			keys = append(keys, node.middleNode.key)
			children = append(children, node.middleNode.child)
		default:
			// If root is leafNode
			keys = append(keys, node.middleNode.key)
			children = append(children, node.middleNode)
		}

		for index, key := range keys {
			child := children[index]
			newKey := node.key

			if !bytes.Equal(key, []byte{0}) {
				newKey = append(node.key, key...)
			}

			if child.NodeType() == TRIE_FULL_NODE ||
				child.NodeType() == TRIE_SHORT_NODE {
				if !bytes.HasPrefix(newKey, iterator.prefix) &&
					!bytes.HasPrefix(iterator.prefix, newKey) {
					continue
				}

				iterator.middleNodes = append(iterator.middleNodes, &middleKeyAndNode{
					key:        newKey,
					middleNode: child,
				})
			} else {
				if !bytes.HasPrefix(newKey, iterator.prefix) {
					continue
				}

				iterator.leafNodes = append(iterator.leafNodes, &leafKeyAndNode{
					key:      newKey,
					leafNode: child,
				})
			}
		}
	}
}
