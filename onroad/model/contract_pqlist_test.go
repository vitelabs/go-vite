package model

import (
	"container/heap"
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

func TestContractBlocks_FindLocationLess(t *testing.T) {
	intSlice := make([]int, 0)
	insertSlice := make([]int, 0)
	intSlice = append(intSlice, 2, 5, 8, 9, 10, 11, 14, 17, 18, 20, 24, 26)
	insertSlice = append(insertSlice, 1, 3, 12, 15, 19, 27)
	fmt.Println(intSlice)

	for k, v := range insertSlice {
		loc := findLocationLess(intSlice, 0, len(intSlice), v)
		fmt.Printf("k:%v v:%v loc:%v\n", k, v, loc)
		addElem(&intSlice, loc, v)
		fmt.Println(intSlice)
		removeElem(&intSlice, loc)
		fmt.Println(intSlice)
	}
}

func addElem(intSlice *[]int, loc int, elem int) {
	(*intSlice) = append((*intSlice)[:loc+1], (*intSlice)[loc:]...)
	(*intSlice)[loc] = elem
}

func findLocationLess(slice []int, low, high int, elem int) int {
	mid := (high-low)/2 + low
	if low >= high {
		return high
	}

	if elem >= slice[mid] {
		return findLocationLess(slice, mid+1, high, elem)
	} else {
		return findLocationLess(slice, low, mid, elem)
	}
	return -1
}

func removeElem(intSlice *[]int, loc int) {
	if len(*intSlice) >= 0 {
		(*intSlice) = append((*intSlice)[:loc], (*intSlice)[loc+1:]...)
	}
}

func TestContractBlocksPQueue_Sort(t *testing.T) {
	var cbpq contractBlocksPQueue
	b1 := &contractBlock{
		block:  ledger.AccountBlock{},
		Height: 8,
	}
	heap.Push(&cbpq, b1)
	b2 := &contractBlock{
		block:  ledger.AccountBlock{},
		Height: 3,
	}
	heap.Push(&cbpq, b2)
	b3 := &contractBlock{
		block:  ledger.AccountBlock{},
		Height: 9,
	}
	heap.Push(&cbpq, b3)
	b4 := &contractBlock{
		block:  ledger.AccountBlock{},
		Height: 6,
	}
	heap.Push(&cbpq, b4)
	for cbpq.Len() > 0 {
		if b, ok := heap.Pop(&cbpq).(*contractBlock); ok {
			t.Logf("%v\n", b.Height)
		}
	}
}
