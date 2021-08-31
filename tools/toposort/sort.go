package toposort

import "sort"

type swapInterface interface {
	Swap(i, j int)
}

type Interface interface {
	swapInterface
	Ids(i int) []string
	Inputs(i int) []string
	Len() int
}

type node struct {
	id     string
	alias  []string
	inputs []string

	index int

	inputCnt    int
	outputs     []string
	sortedIndex int
}

func TopoSort(data Interface) error {
	alias := make(map[string]string)

	len := data.Len()
	unsorted := make([]*node, len)
	for i := 0; i < len; i++ {
		ids := data.Ids(i)
		inputs := data.Inputs(i)
		unsorted[i] = &node{
			id:          ids[0],
			alias:       ids,
			inputs:      inputs,
			index:       i,
			inputCnt:    0,
			sortedIndex: 0,
		}
	}
	for _, n := range unsorted {
		for _, id := range n.alias {
			alias[id] = n.id
		}
	}
	unsortedMap := make(map[string]*node, len)

	for _, n := range unsorted {
		unsortedMap[n.id] = n
	}

	for _, n := range unsorted {
		for _, input := range n.inputs {
			real, ok := alias[input]
			if ok {
				n.inputCnt = n.inputCnt + 1
				unsortedMap[real].outputs = append(unsortedMap[real].outputs, n.id)
			}
		}
	}

	graph := newGraph(unsortedMap)

	sorted, err := graph.tpSort()
	if err != nil {
		return err
	}
	for i, n := range sorted {
		n.sortedIndex = i
	}

	var values []int
	for _, ele := range unsorted {
		values = append(values, ele.sortedIndex)
	}
	wrap := &sortWrap{
		data:   data,
		values: values,
	}
	sort.Sort(wrap)
	return nil
}

type sortWrap struct {
	data   swapInterface
	values []int
}

func (a *sortWrap) Len() int { return len(a.values) }
func (a *sortWrap) Swap(i, j int) {
	a.data.Swap(i, j)
	a.values[i], a.values[j] = a.values[j], a.values[i]
}
func (a *sortWrap) Less(i, j int) bool {
	return a.values[i] < a.values[j]
}
