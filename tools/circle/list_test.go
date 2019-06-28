package circle

import (
	"fmt"
	"testing"
	"time"
)

func TestList_Put(t *testing.T) {
	l := NewList(3)
	for i := 1; i < 4; i++ {
		if old := l.Put(i); old != nil {
			t.Fail()
		}
	}

	if l.Size() != 3 {
		t.Fail()
	}

	old := l.Put(4)
	if v := old.(int); v != 1 {
		t.Fail()
	}

	old = l.Put(5)
	if v := old.(int); v != 2 {
		t.Fail()
	}

	if l.Size() != 3 {
		t.Fail()
	}
}

func TestList_Size(t *testing.T) {
	l := NewList(3)

	if l.Size() != 0 {
		t.Fail()
	}

	for i := 1; i < 4; i++ {
		l.Put(i)
		if l.Size() != i {
			t.Fail()
		}
	}

	for i := 1; i < 4; i++ {
		l.Put(i)
		if l.Size() != 3 {
			t.Fail()
		}
	}
}

func TestList_Traverse(t *testing.T) {
	const total = 3
	l := NewList(total)
	const count = 10
	for i := 1; i < count; i++ {
		l.Put(i)
	}

	start := count - total
	var value int
	l.Traverse(func(key Key) bool {
		if value = key.(int); value != start {
			t.Fail()
		}
		start++
		return true
	})

	if value != count-1 {
		t.Fail()
	}
}

func TestList_TraverseR(t *testing.T) {
	const total = 3
	l := NewList(total)
	const count = 10
	for i := 1; i < count; i++ {
		l.Put(i)
	}

	fmt.Printf("%+v\n", l)

	start := count - 1
	var value int
	l.TraverseR(func(key Key) bool {
		fmt.Println(key)
		if value = key.(int); value != start {
			t.Fail()
		}
		start--
		return true
	})

	if value != count-total {
		t.Fail()
	}
}

func TestList_Reset(t *testing.T) {
	const total = 3
	l := NewList(total)
	const count = 10
	for i := 1; i < count; i++ {
		l.Put(i)
	}

	fmt.Printf("%+v\n", l)
	l.Reset()
	fmt.Printf("%+v\n", l)

	var k int
	l.Traverse(func(key Key) bool {
		fmt.Println(key)
		k = key.(int)
		return true
	})

	if k != 0 {
		t.Fail()
	}
}

func TestList_Put_Concurrent(t *testing.T) {
	const total = 86400
	l := NewList(total)

	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100000; j++ {
				l.Put(j)
			}
		}()
	}

	go func() {
		for {
			var amount int
			l.TraverseR(func(key Key) bool {
				k, ok := key.(int)
				if ok {
					amount += k
				} else {
					panic(fmt.Errorf("invalid value: %v", key))
				}
				return true
			})

			time.Sleep(100 * time.Millisecond)
		}
	}()

	time.Sleep(5 * time.Second)
}
