package toposort

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type items struct {
	ids    []string
	inputs []string
}

type byTop []items

func (a byTop) Len() int              { return len(a) }
func (a byTop) Swap(i, j int)         { a[i], a[j] = a[j], a[i] }
func (a byTop) Ids(i int) []string    { return a[i].ids }
func (a byTop) Inputs(i int) []string { return a[i].inputs }

func TestSort(t *testing.T) {
	nodes := []items{
		{
			ids:    []string{"0", "00"},
			inputs: []string{},
		},
		{
			ids:    []string{"3", "03"},
			inputs: []string{"1"},
		},
		{
			ids:    []string{"1", "01"},
			inputs: []string{"0"},
		},
		{
			ids:    []string{"2", "02"},
			inputs: []string{"01"},
		},
	}

	err := TopoSort(byTop(nodes))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	for _, n := range nodes {
		t.Log(n.ids[0])
	}

	assert.Equal(t, nodes[0].ids[0], "0")
	assert.Equal(t, nodes[1].ids[0], "1")
	assert.Equal(t, nodes[2].ids[0], "2")
	assert.Equal(t, nodes[3].ids[0], "3")
}

func TestCycle(t *testing.T) {
	nodes := []items{
		{
			ids:    []string{"A", "a"},
			inputs: []string{"C"},
		},
		{
			ids:    []string{"B", "b"},
			inputs: []string{"a"},
		},
		{
			ids:    []string{"C", "c"},
			inputs: []string{"b"},
		},
	}

	err := TopoSort(byTop(nodes))
	if err == nil {
		t.Errorf("Expected cycle error")
		t.FailNow()
	}
	assert.Error(t, err, "cycle error A B C ")
}

func TestSortWrap(t *testing.T) {
	intCases := []int{4, 2, 1, 5, 3, 6, 0, 7, 8}

	intCasesCopy := make([]int, len(intCases))

	copy(intCasesCopy, intCases)

	result := &sortWrap{
		data:   sort.IntSlice(intCases),
		values: intCasesCopy,
	}

	sort.Sort(result)

	for i, item := range intCases {
		t.Log(item)
		assert.Equal(t, i, item)
	}
	for i, item := range intCasesCopy {
		t.Log(item)
		assert.Equal(t, i, item)
	}
}
