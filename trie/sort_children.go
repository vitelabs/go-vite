package trie

import "sort"

type children struct {
	Key   byte
	Value *TrieNode
}
type sortedChildren []*children

func (a sortedChildren) Len() int           { return len(a) }
func (a sortedChildren) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortedChildren) Less(i, j int) bool { return a[i].Key < a[j].Key }

func newSortedChildren(c map[byte]*TrieNode) sortedChildren {
	s := sortedChildren{}
	for key, child := range c {
		s = append(s, &children{
			Key:   key,
			Value: child,
		})
	}

	sort.Sort(s)
	return s
}
