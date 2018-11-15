package list

import (
	"fmt"
	"testing"
)

func TestList_Shift(t *testing.T) {
	l := New()

	var i int

	const total = 30
	const batch = 10

	for i = 0; i < total; i++ {
		l.Append(i)
		fmt.Println("size", l.Size())
	}

	var e interface{}
	for i, e = 0, l.Shift(); e != nil; e = l.Shift() {
		fmt.Println(e.(int), "size", l.Size())
		i++

		if i >= batch {
			break
		}
	}

	fmt.Println(l.Size())
	if l.Size() != (total-batch) || i != batch {
		t.Fail()
	}
}

func TestList_UnShift(t *testing.T) {
	l := New()

	l.UnShift(1)
	l.UnShift(2)

	var i, j int
	l.Traverse(func(prev, current *Element) {
		i++
		j = current.Value.(int)
	})

	if i != 2 || l.Size() != 2 {
		t.Fail()
	}

	if j != 1 {
		t.Fail()
	}

	i = l.Shift().(int)
	if i != 2 {
		t.Fail()
	}
}
