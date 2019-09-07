package list

import (
	"fmt"
	"testing"
)

func TestList_Append(t *testing.T) {
	l := New()

	const total = 10
	for i := 0; i < total; i++ {
		l.Append(i)
	}

	if l.Size() != total {
		t.Fail()
	}

	i := 0
	l.Traverse(func(value interface{}) bool {
		if v := value.(int); v != i {
			t.Fail()
		}
		i++
		return true
	})
}

func TestList_Shift(t *testing.T) {
	l := New()
	l.Shift()

	const total = 30
	const batch = 10

	for i := 0; i < total; i++ {
		l.Append(i)
	}

	for i := 0; i < batch; i++ {
		e := l.Shift()
		if i != e.(int) {
			fmt.Println("element", i, e)
			t.Fail()
		}
	}

	if l.Size() != (total - batch) {
		fmt.Printf("size %d should be %d\n", l.Size(), total-batch)
		t.Fail()
	}

	for i := total; i < total+batch; i++ {
		l.Append(i)
	}

	// shift all elements
	for j, v := batch, l.Shift(); v != nil; j, v = j+1, l.Shift() {
		if v.(int) != j {
			t.Fail()
		}
	}

	if l.Size() != 0 {
		fmt.Println("size should be 0", l.Size())
		t.Fail()
	}

	// append again
	for i := 0; i < total; i++ {
		l.Append(i)
	}

	for i := 0; i < batch; i++ {
		e := l.Shift()
		if i != e.(int) {
			fmt.Println("element", i, e)
			t.Fail()
		}
	}
}

func TestList_UnShift(t *testing.T) {
	l := New()

	const total = 30
	const batch = 10

	for i := 0; i < total; i++ {
		l.UnShift(i)
	}

	if l.Size() != total {
		t.Fail()
	}

	for i := 1; i <= batch; i++ {
		e := l.Shift()
		if e.(int) != total-i {
			t.Fail()
		}
	}
}

func TestList_Traverse(t *testing.T) {
	l := New()
	l.Traverse(func(value interface{}) bool {
		return true
	})

	const total = 30

	for i := 0; i < total; i++ {
		l.Append(i)
	}

	i := 0
	l.Traverse(func(value interface{}) bool {
		if value.(int) != i {
			fmt.Println("traverse fail")
			t.Fail()
		}
		i++
		return true
	})
}

func TestList_Filter(t *testing.T) {
	l := New()
	l.Filter(func(v interface{}) bool {
		return true
	})

	const total = 30

	for i := 0; i < total; i++ {
		l.Append(i)
	}

	// remove some elements
	const threshold = 10
	l.Filter(func(value interface{}) bool {
		return value.(int) < threshold
	})

	if l.Size() != total-threshold {
		fmt.Println("remove fail")
		t.Fail()
	}

	// append again
	for i := total; i < total+threshold; i++ {
		l.Append(i)
	}

	// rest elements
	i := threshold
	l.Traverse(func(value interface{}) bool {
		if value.(int) != i {
			fmt.Println("rest fail")
			t.Fail()
		}
		i++
		return true
	})

	// remove all elements
	l.Filter(func(value interface{}) bool {
		return true
	})
	if l.Size() != 0 {
		fmt.Println("remove all fail")
		t.Fail()
	}

	// append again
	for i := 0; i < total; i++ {
		l.Append(i)
	}
	i = 0
	l.Traverse(func(value interface{}) bool {
		if value.(int) != i {
			t.Fail()
		}
		i++
		return true
	})
}
