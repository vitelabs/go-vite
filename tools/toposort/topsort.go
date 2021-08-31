package toposort

import (
	"sort"

	"github.com/pkg/errors"
)

type byId []*node

func (p byId) Len() int           { return len(p) }
func (p byId) Less(i, j int) bool { return p[i].id < p[j].id }
func (p byId) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type graph struct {
	sorted   []*node
	unsorted map[string]*node
}

// type node struct {
// 	id     string
// 	inputs int
// 	refs   map[string]*node
// }

func (g graph) filterZero() []*node {
	var result []*node
	for _, n := range g.unsorted {
		if n.inputCnt == 0 {
			result = append(result, n)
		}
	}
	sort.Sort(byId(result))
	return result
}

func (g *graph) move(n *node) {
	g.sorted = append(g.sorted, n)
	delete(g.unsorted, n.id)
}

func (g graph) unsortedResult() string {
	var ids []string
	for _, ele := range g.unsorted {
		ids = append(ids, ele.id)
	}
	var result string
	sort.Strings(ids)
	for _, ele := range ids {
		result += ele + " "
	}
	return result
}

func (g *graph) tpSort() ([]*node, error) {
	for {
		waitings := g.filterZero()

		if len(waitings) == 0 && len(g.unsorted) != 0 {
			return nil, errors.Errorf("cycle error %s", g.unsortedResult())
		}

		for _, n := range waitings {
			for _, output := range n.outputs {
				refNode := g.unsorted[output]
				refNode.inputCnt = refNode.inputCnt - 1
			}
			g.move(n)
		}
		if len(g.unsorted) == 0 {
			break
		}
	}
	return g.sorted, nil
}

func newGraph(nodes map[string]*node) *graph {
	result := &graph{
		sorted:   []*node{},
		unsorted: nodes,
	}
	return result
}
