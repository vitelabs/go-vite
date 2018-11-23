package trie

type NodeIterator struct {
	currentNode *TrieNode

	tmpNodes []*TrieNode
}

func NewNodeIterator(trie *Trie) *NodeIterator {
	ni := &NodeIterator{
		tmpNodes: []*TrieNode{trie.Root},
	}

	return ni
}

func (ni *NodeIterator) Next(deepInto func(*TrieNode) bool) bool {
	if len(ni.tmpNodes) <= 0 {
		return false
	}
	node := ni.tmpNodes[0]
	ni.currentNode = node

	ni.tmpNodes = ni.tmpNodes[1:]
	if deepInto(node) {
		switch node.NodeType() {
		case TRIE_FULL_NODE:
			for _, child := range node.children {
				ni.tmpNodes = append(ni.tmpNodes, child)
			}
		case TRIE_SHORT_NODE:
			ni.tmpNodes = append(ni.tmpNodes, node.child)
		}
	}

	return true
}

func (ni *NodeIterator) Node() *TrieNode {
	return ni.currentNode
}
